package amqp

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	log "github.com/Sirupsen/logrus"
	"github.com/getconversio/go-utils/util"
	"github.com/streadway/amqp"
)

var (
	conn               *amqp.Connection
	ch                 *amqp.Channel
	consumerSeq        uint64
	channelOnce        sync.Once
	retryConsumerOnce  sync.Once
	queueTtls          = []int{1, 5, 10, 30, 60, 300, 600}
	readyQueueName     = util.Getenv("RABBITMQ_READY_QUEUE", "amqp.retry.ready")
	retryExchange      = util.Getenv("RABBITMQ_RETRY_EXCHANGE", "amqp.retry")
	retryRoutingKey    = util.Getenv("RABBITMQ_RETRY_ROUTING", "retry")
	retryQueueTemplate = util.Getenv("RABBITMQ_RETRY_QUEUE", "amqp.retry.waiting") + "-%04d"
)

type EmptyCreator interface {
	NewEmpty() interface{}
}

func onCancel() {
	if os.Getenv("RABBITMQ_CLOSE_ON_CANCEL") == "yes" {
		conn.Close()
	}
}

func onClose() {
	if os.Getenv("RABBITMQ_EXIT_ON_CLOSE") == "yes" {
		log.Panic("AMQP connection closed, don't know what to")
	}
}

func newChannel() (*amqp.Connection, *amqp.Channel) {
	u := os.Getenv("RABBITMQ_URL")

	// If the url is empty, try a provider specific url.
	if u == "" {
		u = os.Getenv("CLOUDAMQP_URL")
	}

	conn, err := amqp.Dial(u)
	util.PanicOnError("Failed to connect to RabbitMQ", err)

	ch, err := conn.Channel()
	util.PanicOnError("Failed to open RabbitMQ channel", err)

	// XXX: Changing the global parameter to true will fail on Codeship because
	// it uses an older version of RabbitMQ that does not support it. So it's
	// set to false by default for now.
	err = ch.Qos(20, 0, false)
	util.PanicOnError("Failed to set Qos on RabbitMQ channel", err)

	return conn, ch
}

func ensureChannel() {
	channelOnce.Do(func() {
		conn, ch = newChannel()

		go func() {
			closeChannel := conn.NotifyClose(make(chan *amqp.Error))
			for e := range closeChannel {
				log.Infof("AMQP received close message: %v", e)
			}
			onClose()
		}()

		_, err := ch.QueueDeclare(
			readyQueueName, // name
			true,           // durable
			false,          // delete when unused
			false,          // exclusive
			false,          // no-wait
			nil,            // arguments
		)
		util.PanicOnError("Failed to declare RabbitMQ queue", err)

		err = ch.ExchangeDeclare(
			retryExchange, // name
			"topic",       // type
			true,          // durable
			false,         // auto-deleted
			false,         // internal
			false,         // no-wait
			nil,           // arguments
		)
		util.PanicOnError("Failed to declare RabbitMQ exchange", err)

		err = ch.QueueBind(
			readyQueueName,  // queue name
			retryRoutingKey, // routing key
			retryExchange,   // exchange name
			false,           // no-wait
			nil,             // arguments
		)
		util.PanicOnError("Failed to bind RabbitMQ queue", err)

		for _, ttl := range queueTtls {
			args := make(amqp.Table)
			args["x-dead-letter-exchange"] = retryExchange
			args["x-dead-letter-routing-key"] = retryRoutingKey
			args["x-message-ttl"] = int32(ttl * 1000)

			_, err = ch.QueueDeclare(
				fmt.Sprintf(retryQueueTemplate, ttl), // name
				true,  // durable
				false, // delete when unused
				false, // exclusive
				false, // no-wait
				args,  // arguments
			)
			util.PanicOnError("Failed to declare RabbitMQ queue", err)
		}
	})
}

// Ensures that the exchange with the given name exists.
// It is not necessary to call this function when using HandleFunc
func EnsureExchange(exchangeName string) {
	ensureChannel()

	err := ch.ExchangeDeclare(
		exchangeName,
		"topic", // type
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	)
	util.PanicOnError("Failed to declare RabbitMQ exchange", err)
}

// Ensures that the queue with the given name exists.
// It is not necessary to call this function when using HandleFunc
func EnsureQueue(queueName string) {
	ensureChannel()

	_, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	util.PanicOnError("Failed to declare RabbitMQ queue", err)
}

func PurgeQueue(queueName string) {
	_, err := ch.QueuePurge(queueName, false)
	util.PanicOnError("Failed to purge RabbitMQ queue", err)
}

// A raw message struct. Used by the retry handler.
// Note: while encoding/json has a RawMessage type that works essentially the
// same way, it did not work well for this use case.
type retryCarrier struct {
	Raw []byte
}

func (t *retryCarrier) NewEmpty() interface{} {
	return new(retryCarrier)
}

func (t *retryCarrier) UnmarshalJSON(data []byte) error {
	t.Raw = data
	return nil
}

func (t *retryCarrier) MarshalJSON() ([]byte, error) {
	return t.Raw, nil
}

func EnsureRetryConsumer() {
	retryConsumerOnce.Do(func() {
		ensureChannel()

		HandleFunc(readyQueueName, retryExchange, retryRoutingKey, new(retryCarrier), func(msg interface{}, headers amqp.Table) error {
			retryNumber, _ := strconv.Atoi(headers["_retryNumber"].(string))
			if retryNumber >= len(queueTtls) {
				log.Error("Permanent task failure")
				return nil
			}

			exchangeName := headers["_exchangeName"].(string)
			routingKey := headers["_routingKey"].(string)

			msgBytes, err := json.Marshal(msg)
			util.PanicOnError("Failed to marshall JSON data for RabbitMQ message", err)

			return ch.Publish(
				exchangeName,
				routingKey,
				false, // Mandatory
				false, // Immediate
				amqp.Publishing{
					ContentType: "application/json",
					Body:        msgBytes,
					Headers:     headers,
				})
		})
	})
}

func Publish(exchangeName, routingKey string, msg interface{}) error {
	ensureChannel()

	msgBytes, err := json.Marshal(msg)
	util.PanicOnError("Failed to marshall JSON data for RabbitMQ message", err)

	return ch.Publish(
		exchangeName,
		routingKey,
		false, // Mandatory
		false, // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msgBytes,
		})
}

func publishRetry(message amqp.Delivery) error {
	retryNumber := 0

	if len(message.Headers) == 0 {
		message.Headers = make(amqp.Table)
	}

	message.Headers["_exchangeName"] = message.Exchange
	message.Headers["_routingKey"] = message.RoutingKey

	if r, ok := message.Headers["_retryNumber"]; ok {
		retryNumber, _ = strconv.Atoi(r.(string))
	}

	if retryNumber >= len(queueTtls) {
		return nil
	}

	// TODO test this
	message.Headers["_retryNumber"] = strconv.Itoa(retryNumber + 1)

	return ch.Publish(
		"",
		fmt.Sprintf(retryQueueTemplate, queueTtls[retryNumber]),
		false, // Mandatory
		false, // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message.Body,
			Headers:     message.Headers,
		})
}

// Sets up a handler function for the given queue, exchange and routing key.
// The handlerMsg interface is needed in order for the returned handler to be
// able to cast the returned message into the right type.
func HandleFunc(queueName, exchangeName, routingKey string, msgCreator EmptyCreator, handler func(interface{}, amqp.Table) error) string {
	ensureChannel()
	EnsureExchange(exchangeName)
	EnsureQueue(queueName)

	err := ch.QueueBind(queueName, routingKey, exchangeName, false, nil)
	util.PanicOnError("Failed to bind RabbitMQ queue", err)

	// Inspired by the amqp code
	ctag := fmt.Sprintf("ctag-%d", atomic.AddUint64(&consumerSeq, 1))

	// Consume from a queue and create a go-routine that listens for messages
	// forever.
	msgs, err := ch.Consume(
		queueName, // queue
		ctag,      // consumer tag
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	util.PanicOnError("Failed to register a RabbitMQ consumer", err)

	logger := log.WithFields(log.Fields{
		"ctag":     ctag,
		"queue":    queueName,
		"exchange": exchangeName,
		"routing":  routingKey,
	})

	shouldRetry := true
	// If we're setting up the ready queue, it should never retry from failures on that queue...
	if queueName == readyQueueName {
		shouldRetry = false
	}

	go func() {
		for msg := range msgs {
			// Create a new empty handlerMsg
			handlerMsg := msgCreator.NewEmpty()

			// Assume JSON and unmarshal the body of the message into the given handler message.
			err = json.Unmarshal(msg.Body, handlerMsg)

			// An error when unmarshalling the JSON is not something we can
			// retry. Log an error and ack the message.
			if err != nil {
				logger.WithField("body", fmt.Sprintf("%s", msg.Body)).Errorf("Could not unmarshal AMQP message: %s", err)
				msg.Ack(false)
				continue
			}

			// Run the handler
			err = handler(handlerMsg, msg.Headers)
			if err != nil {
				logger.Errorf("Error while processing message: %s", err)

				if shouldRetry {
					err = publishRetry(msg)

					if err != nil {
						logger.Errorf("Error while trying to publish to retry queue: %s", err)
					}
				}
			}

			// Ack the message
			err = msg.Ack(false)
			if err != nil {
				logger.Errorf("Could not ack message: %s", err)
			}
		}
		logger.Info("AMQP consumer was cancelled")
		onCancel()
	}()

	logger.Debug("Handler waiting for messages")
	return ctag
}

func QueueTotalMessages(queueNames []string) int {
	ensureChannel()

	total := 0
	for _, queueName := range queueNames {
		queue, _ := ch.QueueInspect(queueName)
		total += queue.Messages
	}
	return total
}

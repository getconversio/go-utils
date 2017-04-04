package amqp

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/getconversio/go-utils/util"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	notify     chan amqp.Confirmation
	acks       int
	nacks      int
	total      int
	intChannel chan int
)

type msgType struct {
	I int `json:"i"`
}

func (m *msgType) NewEmpty() interface{} {
	return new(msgType)
}

var notifyOnce sync.Once
var notifyLock sync.Mutex

func resetAcksNack() {
	notifyLock.Lock()
	acks = 0
	nacks = 0
	notifyLock.Unlock()
}

func resetQueues(t *testing.T) {
	_, err := ch.QueuePurge("amqp.retry.waiting-0001", false)
	require.NoError(t, err)
	_, err = ch.QueuePurge("amqp.retry.waiting-0005", false)
	require.NoError(t, err)
	_, err = ch.QueuePurge("amqp.retry.waiting-0300", false)
	require.NoError(t, err)
	_, err = ch.QueuePurge("amqp.retry.ready", false)
	require.NoError(t, err)
}

func setup() {
	ensureChannel()
	ch.QueueDelete("test.mctest", false, false, false)
	ch.ExchangeDelete("test", false, false)
	resetAcksNack()

	// Start a notification listener so we can keep track of acks and nacks.
	// Only do this once though.
	notifyOnce.Do(func() {
		notify = ch.NotifyPublish(make(chan amqp.Confirmation))
		go func() {
			for n := range notify {
				notifyLock.Lock()
				if n.Ack {
					acks++
				} else {
					nacks++
				}
				notifyLock.Unlock()
			}
		}()
		ch.Confirm(false)
	})
	intChannel = make(chan int)
}

func teardown() {
	ch.QueueDelete("test.mctest", false, false, false)
	ch.ExchangeDelete("test", false, false)
	close(intChannel)
}

func testHandle(f func(interface{}, amqp.Table) error) string {
	return HandleFunc(
		"test.mctest", // Queue name
		"test",        // Exchange name
		"test.mctest", // Routing key
		new(msgType),
		f)
}

func TestHandleFunc(t *testing.T) {
	setup()
	defer teardown()

	cases := []struct {
		fun                func(interface{}, amqp.Table) error
		body               []byte
		headers            amqp.Table
		acksWanted         int
		nacksWanted        int
		retryQueueLength   int
		retriesWantedIn300 int
	}{
		{
			// Wrong JSON is acked
			func(msg interface{}, headers amqp.Table) error { return nil },
			[]byte("not JSON"),
			amqp.Table{},
			1,
			0,
			0,
			0,
		},
		{
			// Error in handler function is acked.
			func(msg interface{}, headers amqp.Table) error { return errors.New("not good") },
			[]byte("{}"),
			amqp.Table{},
			2,
			0,
			1,
			0,
		},
		{
			// Test _retryNumber
			func(msg interface{}, headers amqp.Table) error { return errors.New("not good") },
			[]byte("{}"),
			amqp.Table{
				"_retryNumber": "5",
			},
			2,
			0,
			0,
			1,
		},
		{
			// Test _retryNumber limit
			func(msg interface{}, headers amqp.Table) error { return errors.New("not good") },
			[]byte("{}"),
			amqp.Table{
				"_retryNumber": "7",
			},
			1,
			0,
			0,
			0,
		},
	}

	for _, c := range cases {
		resetAcksNack()
		resetQueues(t)
		ctag := testHandle(c.fun)

		// Force an error
		err := ch.Publish(
			"test",
			"test.mctest",
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        c.body,
				Headers:     c.headers,
			})

		// There should be no publish error
		require.NoError(t, err)

		validator := func() bool {
			notifyLock.Lock()
			actual := acks + nacks
			notifyLock.Unlock()

			return c.acksWanted+c.nacksWanted == actual
		}

		// Wait up to 500 milliseconds for notifications of delivery.
		util.ValidateWithTimeout(t, validator, 500)

		// Assert the number of acks and nacks
		assert.Equal(t, c.acksWanted, acks)
		assert.Equal(t, c.nacksWanted, nacks)

		validator = func() bool {
			queue, err := ch.QueueInspect("amqp.retry.ready")
			require.NoError(t, err)
			return c.retryQueueLength == queue.Messages
		}

		util.ValidateWithTimeout(t, validator, 2000)

		queue, err := ch.QueueInspect("amqp.retry.waiting-0300")
		require.NoError(t, err)

		assert.Equal(t, c.retriesWantedIn300, queue.Messages)

		ch.Cancel(ctag, false)
		resetQueues(t)
	}
}

func TestEnsureRetryConsumer(t *testing.T) {
	setup()
	defer teardown()

	testHandle(func(msg interface{}, headers amqp.Table) error {
		intChannel <- 1
		return nil
	})

	EnsureRetryConsumer()

	headers := amqp.Table{
		"_retryNumber":  "1",
		"_exchangeName": "test",
		"_routingKey":   "test.mctest",
	}

	err := ch.Publish(
		"",
		readyQueueName,
		false, // Mandatory
		false, // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte("{}"),
			Headers:     headers,
		})

	require.NoError(t, err)

	i := <-intChannel
	assert.Equal(t, 1, i)
}

func TestSharedState(t *testing.T) {
	setup()
	defer teardown()

	messages := []msgType{}

	testHandle(func(msg interface{}, headers amqp.Table) error {
		messages = append(messages, *msg.(*msgType))
		intChannel <- 1
		return nil
	})

	err := ch.Publish(
		"test",
		"test.mctest",
		false, // Mandatory
		false, // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(`{"I":123}`),
		})

	require.NoError(t, err)

	<-intChannel
	assert.Len(t, messages, 1)

	err = ch.Publish(
		"test",
		"test.mctest",
		false, // Mandatory
		false, // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(`{}`),
		})

	<-intChannel
	assert.Len(t, messages, 2)

	// The first message has 123.
	assert.Equal(t, 123, messages[0].I)

	// The second message should be initialized to 0.
	// TDD-note: It was 123 before this test because of the shared message
	// container in the handler.
	assert.Equal(t, 0, messages[1].I)
}

func TestCloseOnCancel(t *testing.T) {
	// http://stackoverflow.com/a/33404435
	if os.Getenv("CLOSE_IT") == "yes" {
		err := os.Setenv("RABBITMQ_CLOSE_ON_CANCEL", "yes")
		require.NoError(t, err)
		err = os.Setenv("RABBITMQ_EXIT_ON_CLOSE", "yes")
		require.NoError(t, err)

		ctag := testHandle(func(msg interface{}, headers amqp.Table) error { return nil })
		ch.Cancel(ctag, false) // This should exit the process

		// Allow up to five seconds time for the handler to cancel and close
		time.Sleep(5 * time.Second)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestCloseOnCancel")
	cmd.Env = append(os.Environ(), "CLOSE_IT=yes")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Errorf("process ran with err %v, want exit status 1, %v", err, cmd.Path)
}

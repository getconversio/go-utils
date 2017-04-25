// Package bq provides a wrapper for Google's BigQuery library as well as
// general setup of the BigQuery client and streaming inserts usign bqstreamer.
package bq

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	log "github.com/Sirupsen/logrus"
	"github.com/getconversio/go-utils/util"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/option"
	bqstreamer "gopkg.in/rounds/go-bqstreamer.v2"
)

type BigQueryWrapper struct {
	Client      *bigquery.Client
	ProjectId   string
	DatasetId   string
	TablePrefix string
}

var insertWorker *bqstreamer.AsyncWorkerGroup

func (w *BigQueryWrapper) Dataset() *bigquery.Dataset {
	return w.Client.Dataset(w.DatasetId)
}

func (w *BigQueryWrapper) TableId(tableId string) string {
	if w.TablePrefix != "" {
		tableId = fmt.Sprintf("%s_%s", w.TablePrefix, tableId)
	}
	return tableId
}

func (w *BigQueryWrapper) Table(tableId string) *bigquery.Table {
	tableId = w.TableId(tableId)
	return w.Dataset().Table(tableId)
}

func (w *BigQueryWrapper) AddRow(tableId string, row interface{}) error {
	data, err := EncodeLegacy(row, true)
	if err != nil {
		return err
	}
	log.Debugf("BigQuery mapped row for %s: %#v", tableId, data)
	mappedRow := bqstreamer.NewRow(w.ProjectId, w.DatasetId, w.TableId(tableId), data)
	insertWorker.Enqueue(mappedRow)
	return nil
}

func (w *BigQueryWrapper) UseTablePrefix(useIt bool) {
	if useIt {
		hostname, err := os.Hostname()
		util.PanicOnError("Cannot get hostname", err)
		w.TablePrefix = util.Hash32(hostname)
	} else {
		w.TablePrefix = ""
	}
}

// Creates a new wrapper for a bigquery client
func NewBigQueryWrapper(client *bigquery.Client, projectId, datasetId string) *BigQueryWrapper {
	wrapper := BigQueryWrapper{
		Client:    client,
		ProjectId: projectId,
		DatasetId: datasetId,
	}

	return &wrapper
}

func bigqueryJWTConfig() (*jwt.Config, error) {
	saFile := os.Getenv("BIGQUERY_SERVICE_ACCOUNT")

	if strings.HasSuffix(saFile, ".json") {
		log.Debug("Using JSON file for BigQuery credentials")
		saData, err := ioutil.ReadFile(saFile)
		util.PanicOnError("Cannot read keyfile", err)

		return google.JWTConfigFromJSON(saData, bigquery.Scope)
	} else {
		// saFile is a string, read the string. Assume it's base64 encoded.
		log.Debug("Using configuration value for BigQuery credentials")

		saData, err := base64.StdEncoding.DecodeString(saFile)
		util.PanicOnError("Cannot base64 decode BigQuery credentials", err)

		return google.JWTConfigFromJSON(saData, bigquery.Scope)
	}
}

// Close any open workers.
func Close() {
	if insertWorker != nil {
		log.Info("Waiting for BigQuery insert worker to finish")
		insertWorker.Close()
		insertWorker = nil
	}
}

// Sets up the BigQuery client wrapper. Assumes that the following environment variables are set:
// BIGQUERY_PROJECT_ID: The ID of the bigquery project, from the Google Cloud console
// BIGQUERY_DATASET_ID: The ID of dataset to use.
// BIGQUERY_SERVICE_ACCOUNT: A filepath for a bigquery service account
// configuration OR a base64 encoded string with the service account credentials.
// The created wrapper is returned.
func Setup() *BigQueryWrapper {
	config, err := bigqueryJWTConfig()
	util.PanicOnError("Cannot load BigQuery credentials", err)

	ctx := context.Background()
	opt := option.WithTokenSource(config.TokenSource(ctx))

	client, err := bigquery.NewClient(ctx, os.Getenv("BIGQUERY_PROJECT_ID"), opt)
	util.PanicOnError("Cannot create client for BigQuery", err)

	return NewBigQueryWrapper(client, os.Getenv("BIGQUERY_PROJECT_ID"), os.Getenv("BIGQUERY_DATASET_ID"))
}

func handleInsertError(insertErrs *bqstreamer.InsertErrors) {
	// Each message is a struct that contains zero or more table errors, fetched with All
	// Each table error contain zero or more insert attempts
	// Each insert attempt contain zero or more
	for _, table := range insertErrs.All() {
		for _, attempt := range table.Attempts() {
			// Log insert attempt error.
			if err := attempt.Error(); err != nil {
				log.WithFields(log.Fields{
					"project": attempt.Project,
					"dataset": attempt.Dataset,
					"table":   attempt.Table,
				}).Error("bigquery table insert error", err)
			}

			// Iterate over all rows in attempt.
			for _, row := range attempt.All() {
				// Iterate over all errors in row and log.
				for _, err := range row.All() {
					log.WithFields(log.Fields{
						"project":  attempt.Project,
						"dataset":  attempt.Dataset,
						"table":    attempt.Table,
						"insertid": row.InsertID,
					}).Error("bigquery row insert error", err)
				}
			}
		}
	}
}

// Sets up workers for handling streaming inserts using multiple concurrent go routines.
// Expects the same environment variables as Setup().
func SetupStreamingInserts() {
	config, err := bigqueryJWTConfig()
	util.PanicOnError("Cannot load BigQuery credentials", err)

	// Error handling goroutine
	// bqstreamer sends errors to the error channel.
	errChan := make(chan *bqstreamer.InsertErrors)
	go func() {
		for insertErrs := range errChan {
			handleInsertError(insertErrs)
		}
	}()

	// Initialize a worker group.
	max_retries := util.GetenvInt("BIGQUERY_STREAMING_MAX_RETRIES", 10)
	max_rows := util.GetenvInt("BIGQUERY_STREAMING_MAX_ROWS", 500)
	max_delay := time.Duration(util.GetenvInt("BIGQUERY_STREAMING_MAX_DELAY", 1000))
	insertWorker, err = bqstreamer.NewAsyncWorkerGroup(
		config,
		bqstreamer.SetAsyncNumWorkers(2),                             // Number of background workers in the group.
		bqstreamer.SetAsyncMaxRows(max_rows),                         // Amount of rows that must be enqueued before executing an insert operation to BigQuery.
		bqstreamer.SetAsyncMaxDelay(max_delay*time.Millisecond),      // Time to wait between inserts.
		bqstreamer.SetAsyncRetryInterval(max_delay*time.Millisecond), // Time to wait between failed insert retries.
		bqstreamer.SetAsyncMaxRetries(max_retries),                   // Maximum amount of retries a failed insert is allowed to be retried.
		bqstreamer.SetAsyncIgnoreUnknownValues(false),                // Ignore unknown fields when inserting rows.
		bqstreamer.SetAsyncSkipInvalidRows(false),                    // Skip bad rows when inserting.
		bqstreamer.SetAsyncErrorChannel(errChan),                     // Set unified error channel.
	)

	if err != nil {
		util.PanicOnError("Error setting up BigQuery streaming workers", err)
	}

	insertWorker.Start()
}

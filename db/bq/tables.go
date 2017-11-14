package bq

import (
	"cloud.google.com/go/bigquery"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// Determines whether two BigQuery schemas are the same. Currently does not
// support nested schemas and only checks the name and type of the fields.
func IsSameSchema(s1, s2 bigquery.Schema) bool {
	for _, f1 := range s1 {
		found := false
		for _, f2 := range s2 {
			if f1.Name == f2.Name && f1.Type == f2.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// Ensure that the given table is up-to-date.
// - If the table does not exist, it will be created
// - If the table exists but the schema is outdated, it will be updated
// - Otherwise nothing happens
// The function panics if any of its API calls fail.
func EnsureTable(table *bigquery.Table, schema bigquery.Schema, extraMeta *bigquery.TableMetadata) {
	ctx := context.Background()

	logger := log.WithField("table", table.TableID)
	logger.Debug("Checking table metadata")

	meta, err := table.Metadata(ctx)

	if err != nil {
		logger.Info("Error fetching table. Assuming it does not exist. Creating it")
		if extraMeta != nil {
			meta = extraMeta
		} else {
			meta = &bigquery.TableMetadata{}
		}
		meta.Schema = schema
		if err = table.Create(ctx, meta); err != nil {
			logger.Panic("Could not create BigQuery table", err)
		}
	} else if !IsSameSchema(schema, meta.Schema) {
		logger.Info("Schema is out of date. Updating it")
		update := bigquery.TableMetadataToUpdate{Schema: schema}
		if meta, err = table.Update(ctx, update, meta.ETag); err != nil {
			logger.Panic("Could not update BigQuery table", err)
		}
	}
}

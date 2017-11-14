package bq

import (
	"fmt"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/jarcoal/httpmock.v1"
)

func TestIsSameSchema(t *testing.T) {
	cases := []struct {
		s1   bigquery.Schema
		s2   bigquery.Schema
		want bool
	}{
		{
			bigquery.Schema{},
			bigquery.Schema{},
			true,
		},
		{
			bigquery.Schema{&bigquery.FieldSchema{Name: "a"}},
			bigquery.Schema{&bigquery.FieldSchema{Name: "a"}},
			true,
		},
		{
			bigquery.Schema{&bigquery.FieldSchema{Name: "a"}, &bigquery.FieldSchema{Name: "b"}},
			bigquery.Schema{&bigquery.FieldSchema{Name: "b"}, &bigquery.FieldSchema{Name: "a"}},
			true,
		},
		{
			bigquery.Schema{&bigquery.FieldSchema{Name: "a"}},
			bigquery.Schema{&bigquery.FieldSchema{Name: "b"}},
			false,
		},
	}

	for i, c := range cases {
		got := IsSameSchema(c.s1, c.s2)
		if got != c.want {
			t.Errorf("Got %b, wanted %b, case %d", got, c.want, i)
		}
	}
}

func TestEnsureTableUpdate(t *testing.T) {
	setup()
	defer teardown()

	// Responder for the metadata request for mytable
	tableUrl := "https://www.googleapis.com/bigquery/v2/projects/some-project/datasets/some_dataset/tables/mytable"
	httpmock.RegisterResponder("GET", tableUrl,
		httpmock.NewStringResponder(200, `
			{
			  "kind": "bigquery#table",
			  "etag": "\"hej\"",
			  "id": "some-project:some_dataset.mytable",
			  "tableReference": {
			   "projectId": "some-project",
			   "datasetId": "some_dataset",
			   "tableId": "mytable"
			  },
			  "numBytes": "0",
			  "numLongTermBytes": "0",
			  "numRows": "0",
			  "creationTime": "1480687735761",
			  "expirationTime": "1481292535761",
			  "lastModifiedTime": "1480687735761",
			  "type": "TABLE"
			}`))

	// Responder for the patch operation when table exists but needs updating
	httpmock.RegisterResponder("PATCH", tableUrl,
		httpmock.NewStringResponder(200, "{}"))

	myStruct := struct {
		ID string `bigquery:"id"`
	}{"abc"}

	schema, err := bigquery.InferSchema(myStruct)
	require.NoError(t, err)

	wrapper := Setup()
	require.NotNil(t, wrapper)
	table := wrapper.Table("mytable")

	EnsureTable(table, schema, nil)

	// Ensure that the table was fetched, it determined the schema is wrong, and it updated.
	callInfo := httpmock.GetCallCountInfo()
	getRequest := fmt.Sprintf("GET %s", tableUrl)
	patchRequest := fmt.Sprintf("PATCH %s", tableUrl)
	assert.Equal(t, 1, callInfo[getRequest])
	assert.Equal(t, 1, callInfo[patchRequest])
}

func TestEnsureTableCreate(t *testing.T) {
	setup()
	defer teardown()

	// Responder for the metadata request for mytable
	tableUrl := "https://www.googleapis.com/bigquery/v2/projects/some-project/datasets/some_dataset/tables/mytable"
	httpmock.RegisterResponder("GET", tableUrl,
		httpmock.NewStringResponder(404, `
{
 "error": {
  "errors": [
   {
    "domain": "global",
    "reason": "notFound",
    "message": "Not found: Table some-project:some_dataset.mytable"
   }
  ],
  "code": 404,
  "message": "Not found: Table some-project:some_dataset.mytable"
 }
}
			`))

	tablesUrl := "https://www.googleapis.com/bigquery/v2/projects/some-project/datasets/some_dataset/tables"
	httpmock.RegisterResponder("POST", tablesUrl,
		httpmock.NewStringResponder(200, "{}"))

	myStruct := struct {
		ID string `bigquery:"id"`
	}{"abc"}

	schema, err := bigquery.InferSchema(myStruct)
	require.NoError(t, err)

	wrapper := Setup()
	require.NotNil(t, wrapper)
	table := wrapper.Table("mytable")

	EnsureTable(table, schema, nil)
	assert.NoError(t, err)

	// Ensure that the table was fetched, it determined there was no table, and it created the table.
	callInfo := httpmock.GetCallCountInfo()
	getRequest := fmt.Sprintf("GET %s", tableUrl)
	postRequest := fmt.Sprintf("POST %s", tablesUrl)
	assert.Equal(t, 1, callInfo[getRequest])
	assert.Equal(t, 1, callInfo[postRequest])
}

func TestEnsureTablePanic(t *testing.T) {
	setup()
	defer teardown()

	// Responder for the metadata request for mytable
	tableUrl := "https://www.googleapis.com/bigquery/v2/projects/some-project/datasets/some_dataset/tables/mytable"
	httpmock.RegisterResponder("GET", tableUrl,
		httpmock.NewStringResponder(404, `
{
 "error": {
  "errors": [
   {
    "domain": "global",
    "reason": "notFound",
    "message": "Not found: Table some-project:some_dataset.mytable"
   }
  ],
  "code": 404,
  "message": "Not found: Table some-project:some_dataset.mytable"
 }
}
			`))

	tablesUrl := "https://www.googleapis.com/bigquery/v2/projects/some-project/datasets/some_dataset/tables"
	httpmock.RegisterResponder("POST", tablesUrl,
		httpmock.NewStringResponder(400, "{}"))

	myStruct := struct {
		ID string `bigquery:"id"`
	}{"abc"}

	schema, err := bigquery.InferSchema(myStruct)
	require.NoError(t, err)

	wrapper := Setup()
	require.NotNil(t, wrapper)
	table := wrapper.Table("mytable")

	assert.Panics(t, func() { EnsureTable(table, schema, nil) })
}

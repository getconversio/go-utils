package bq

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/bigquery/v2"
)

type TestStruct struct {
	A        string
	B        string `bigquery:"foo"`
	C        string `bigquery:"bar"`
	Excluded string `bigquery:"-"`
}

type EmbeddedStruct struct {
	D string
	TestStruct
}

type NestedStruct struct {
	Field  string
	Nested TestStruct
}

type TimeStruct struct {
	Field string
	Time  time.Time
}

type PointerStruct struct {
	A *int
	B *int `bigquery:"foo"`
}

func TestEncodeLegacy(t *testing.T) {
	i1 := 123
	i2 := 456

	cases := []struct {
		in        interface{}
		expected  map[string]bigquery.JsonValue
		omitEmpty bool
	}{
		{
			TestStruct{
				A: "hello",
				B: "world",
				C: "it works",
			},
			map[string]bigquery.JsonValue{
				"A":   bigquery.JsonValue("hello"),
				"foo": bigquery.JsonValue("world"),
				"bar": bigquery.JsonValue("it works"),
			},
			true,
		},
		{
			TestStruct{
				A: "hello",
				B: "world",
			},
			map[string]bigquery.JsonValue{
				"A":   bigquery.JsonValue("hello"),
				"foo": bigquery.JsonValue("world"),
			},
			true,
		},
		{
			TestStruct{
				A: "hello",
				B: "world",
			},
			map[string]bigquery.JsonValue{
				"A":   bigquery.JsonValue("hello"),
				"foo": bigquery.JsonValue("world"),
				"bar": bigquery.JsonValue(""),
			},
			false,
		},
		{
			EmbeddedStruct{
				D: "single",
				TestStruct: TestStruct{
					A: "embedded",
					B: "struct",
					C: "also works",
				},
			},
			map[string]bigquery.JsonValue{
				"A":   bigquery.JsonValue("embedded"),
				"foo": bigquery.JsonValue("struct"),
				"bar": bigquery.JsonValue("also works"),
				"D":   bigquery.JsonValue("single"),
			},
			true,
		},
		{
			NestedStruct{
				Field: "single",
				Nested: TestStruct{
					A: "nested",
					B: "struct",
					C: "also works",
				},
			},
			map[string]bigquery.JsonValue{
				"Field": bigquery.JsonValue("single"),
				"Nested": map[string]bigquery.JsonValue{
					"A":   bigquery.JsonValue("nested"),
					"foo": bigquery.JsonValue("struct"),
					"bar": bigquery.JsonValue("also works"),
				},
			},
			true,
		},
		{
			TimeStruct{
				Field: "my field value",
				Time:  time.Date(2016, 12, 29, 0, 0, 0, 0, time.UTC),
			},
			map[string]bigquery.JsonValue{
				"Field": bigquery.JsonValue("my field value"),
				"Time":  bigquery.JsonValue(time.Date(2016, 12, 29, 0, 0, 0, 0, time.UTC)),
			},
			true,
		},
		{
			TimeStruct{
				Field: "my field value",
			},
			map[string]bigquery.JsonValue{
				"Field": bigquery.JsonValue("my field value"),
			},
			true,
		},
		{
			PointerStruct{},
			map[string]bigquery.JsonValue{
				"A":   bigquery.JsonValue(nil),
				"foo": bigquery.JsonValue(nil),
			},
			false,
		},
		{
			PointerStruct{},
			map[string]bigquery.JsonValue{},
			true,
		},
		{
			PointerStruct{
				A: &i1,
				B: &i2,
			},
			map[string]bigquery.JsonValue{
				"A":   bigquery.JsonValue(123),
				"foo": bigquery.JsonValue(456),
			},
			true,
		},
	}

	for _, c := range cases {
		out, err := EncodeLegacy(c.in, c.omitEmpty)
		assert.NoError(t, err)
		assert.Equal(t, c.expected, out)
	}
}

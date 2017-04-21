package bq

import (
	"fmt"
	"reflect"
	"time"

	"google.golang.org/api/bigquery/v2"
)

// Encode takes a struct and returns a BigQuery compatible encoded map for the
// legacy BigQuery library.
func EncodeLegacy(v interface{}, omitEmpty bool) (map[string]bigquery.JsonValue, error) {
	value := reflect.ValueOf(v)

	if value.Kind() != reflect.Struct {
		if value.Kind() == reflect.Ptr {
			return EncodeLegacy(value.Elem().Interface(), omitEmpty)
		}
		return nil, fmt.Errorf("bigquery: unsupported type: %T (%v)", v, value.Kind())
	}

	m := make(map[string]bigquery.JsonValue)

	valueType := value.Type()
	for i := 0; i < valueType.NumField(); i++ {
		field := valueType.Field(i)
		fieldValue := value.Field(i)

		name := fieldInfo(field)
		switch {
		case !fieldValue.CanInterface(), name == "-":
			continue
		}

		var fieldValueInterface interface{}
		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				fieldValueInterface = nil
			} else {
				fieldValueInterface = fieldValue.Elem().Interface()
			}
		} else {
			fieldValueInterface = fieldValue.Interface()
		}

		if fieldValue.Kind() == reflect.Struct && fieldValue.Type() != reflect.TypeOf(time.Time{}) {
			if mm, err := EncodeLegacy(fieldValueInterface, omitEmpty); err == nil {
				// The fields of an embedded struct gets added directly to the map here
				if field.Anonymous {
					for k, v := range mm {
						m[k] = v
					}
				} else {
					m[name] = mm
				}
			}
		} else {
			if !(omitEmpty &&
				(fieldValueInterface == reflect.Zero(fieldValue.Type()).Interface() ||
					fieldValueInterface == nil)) {
				m[name] = bigquery.JsonValue(fieldValueInterface)
			}
		}
	}

	return m, nil
}

func fieldInfo(field reflect.StructField) string {
	if tag := field.Tag.Get("bigquery"); tag != "" {
		return tag
	}
	return field.Name
}

package service

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

func completeSnapshotV1(data json.RawMessage) bool {
	return completeSnapshotValue(data, reflect.TypeFor[snapshotResumeV1](), false)
}

// Presence comes from the frozen storage schema, not API request binding tags.
// Scalar types and values are subsequently checked by the strict JSON decoder.
func completeSnapshotValue(data json.RawMessage, schema reflect.Type, nullable bool) bool {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nullable
	}
	switch schema.Kind() {
	case reflect.Pointer:
		return completeSnapshotValue(data, schema.Elem(), false)
	case reflect.Slice:
		var entries []json.RawMessage
		if err := json.Unmarshal(data, &entries); err != nil {
			return false
		}
		for _, entry := range entries {
			if !completeSnapshotValue(entry, schema.Elem(), false) {
				return false
			}
		}
		return true
	case reflect.Struct:
		// time.Time is a scalar JSON string, rather than a nested storage object.
		if schema == reflect.TypeFor[time.Time]() {
			return true
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
			return false
		}
		return completeSnapshotFields(fields, schema)
	default:
		return len(bytes.TrimSpace(data)) != 0
	}
}

func completeSnapshotFields(fields map[string]json.RawMessage, schema reflect.Type) bool {
	for i := 0; i < schema.NumField(); i++ {
		field := schema.Field(i)
		if field.Anonymous {
			if !completeSnapshotFields(fields, field.Type.Elem()) {
				return false
			}
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		data, exists := fields[name]
		if !exists {
			return false
		}
		// The only nonembedded pointers in v1 are optional contact/summary.
		// Nil collections also represent no entries; individual entries cannot be nil.
		nullable := field.Type.Kind() == reflect.Pointer || field.Type.Kind() == reflect.Slice
		if !completeSnapshotValue(data, field.Type, nullable) {
			return false
		}
	}
	return true
}

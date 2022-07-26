package nginx

import (
	"fmt"
)

type Fields map[string]string

type LogEntry struct {
	fields Fields
}

func (e *LogEntry) Fields() Fields {
	return e.fields
}

func (e *LogEntry) Field(name string) (value string, err error) {
	value, ok := e.fields[name]
	if !ok {
		err = fmt.Errorf("field '%v' does not found in record %+v", name, *e)
	}
	return
}

func (e *LogEntry) SetField(name, value string) {
	e.fields[name] = value
}

func NewEntry() *LogEntry {
	return &LogEntry{make(Fields)}
}

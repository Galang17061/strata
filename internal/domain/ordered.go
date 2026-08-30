package domain

import (
	"bytes"
	"encoding/json"
)

type OrderedMap struct {
	keys   []string
	values map[string]any
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{values: map[string]any{}}
}

func (m *OrderedMap) Set(key string, value any) {
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = value
}

func (m *OrderedMap) Get(key string) (any, bool) {
	value, ok := m.values[key]
	return value, ok
}

func (m *OrderedMap) Has(key string) bool {
	_, ok := m.values[key]
	return ok
}

func (m *OrderedMap) Keys() []string {
	return append([]string{}, m.keys...)
}

func (m *OrderedMap) Len() int {
	return len(m.keys)
}

func (m *OrderedMap) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	for index, key := range m.keys {
		if index > 0 {
			buffer.WriteString(",")
		}
		name, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(m.values[key])
		if err != nil {
			return nil, err
		}
		buffer.Write(name)
		buffer.WriteString(":")
		buffer.Write(value)
	}
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

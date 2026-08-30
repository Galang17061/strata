package reliability

type orderedMap[V any] struct {
	keys   []string
	values map[string]V
}

func newOrderedMap[V any]() *orderedMap[V] {
	return &orderedMap[V]{values: map[string]V{}}
}

func (m *orderedMap[V]) Has(key string) bool {
	_, ok := m.values[key]
	return ok
}

func (m *orderedMap[V]) Get(key string) V {
	return m.values[key]
}

func (m *orderedMap[V]) Set(key string, value V) {
	if _, ok := m.values[key]; !ok {
		m.keys = append(m.keys, key)
	}
	m.values[key] = value
}

func (m *orderedMap[V]) TryAdd(key string, value V) {
	if _, ok := m.values[key]; !ok {
		m.Set(key, value)
	}
}

func (m *orderedMap[V]) Remove(key string) {
	if _, ok := m.values[key]; !ok {
		return
	}
	delete(m.values, key)
	for index, existing := range m.keys {
		if existing == key {
			m.keys = append(m.keys[:index], m.keys[index+1:]...)
			return
		}
	}
}

func (m *orderedMap[V]) Keys() []string {
	return append([]string{}, m.keys...)
}

func (m *orderedMap[V]) Len() int {
	return len(m.keys)
}

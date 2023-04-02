package store

import (
	"github.com/anacrolix/dht/v2/bep44"
	"sync"
)

var _ bep44.Store = &Memory{}

type Memory struct {
	mu sync.RWMutex
	m  map[bep44.Target]*bep44.Item
}

func NewMemory() *Memory {
	return &Memory{
		m: make(map[bep44.Target]*bep44.Item),
	}
}

func (m *Memory) Put(i *bep44.Item) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.m[i.Target()] = i

	return nil
}

func (m *Memory) Get(t bep44.Target) (*bep44.Item, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	i, ok := m.m[t]
	if !ok {
		return nil, bep44.ErrItemNotFound
	}

	return i, nil
}

func (m *Memory) Del(t bep44.Target) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.m, t)

	return nil
}

func (m *Memory) GetAll() map[bep44.Target]*bep44.Item {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ret := make(map[bep44.Target]*bep44.Item, len(m.m))
	for key, value := range m.m {
		ret[key] = value
	}
	return ret
}

package fab

import "sync"

type registry[T any] struct {
	mu    sync.Mutex
	items map[string]T
}

func newRegistry[T any]() *registry[T] {
	return &registry[T]{items: make(map[string]T)}
}

func (r *registry[T]) add(name string, val T) {
	r.mu.Lock()
	r.items[name] = val
	r.mu.Unlock()
}

func (r *registry[T]) lookup(name string) (T, bool) {
	r.mu.Lock()
	val, ok := r.items[name]
	r.mu.Unlock()
	return val, ok
}

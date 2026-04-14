package lazy

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type (
	Loader[T any] interface {
		MustLoad() T
		Load() (T, error)
		IfLoaded(func(T))
	}

	DeferredLoader[T any] interface {
		Loader[T]
		Set(func() (T, error))
	}

	loader[T any] struct {
		provider func() (T, error)
		onceLoad *sync.Once
		isLoaded atomic.Bool
		value    T
		err      error
	}
)

func New[T any](provider func() (T, error)) Loader[T] {
	impl := NewDeferred[T]()
	impl.Set(provider)
	return impl
}

func NewDeferred[T any]() DeferredLoader[T] {
	var empty T
	return &loader[T]{
		provider: func() (T, error) {
			return empty, fmt.Errorf("provider for %T is not set", empty)
		},
		onceLoad: &sync.Once{},
		isLoaded: atomic.Bool{},
		value:    empty,
		err:      nil,
	}
}

func (l *loader[T]) MustLoad() T {
	value, err := l.Load()
	if err != nil {
		panic(err)
	}

	return value
}

func (l *loader[T]) Load() (T, error) {
	l.onceLoad.Do(func() {
		value, err := l.provider()
		if err != nil {
			l.err = fmt.Errorf("load value of %T: %w", l.value, err)
			return
		}

		l.value = value
		l.isLoaded.Store(true)
	})

	return l.value, l.err
}

func (l *loader[T]) Set(f func() (T, error)) {
	l.provider = f
}

func (l *loader[T]) IfLoaded(f func(T)) {
	if l.isLoaded.Load() {
		f(l.value)
	}
}

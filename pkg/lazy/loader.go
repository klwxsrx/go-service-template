package lazy

import (
	"fmt"
	"sync"
)

type Loader[T any] interface {
	MustLoad() T
	Load() (T, error)
	IfLoaded(func(T))
}

type loader[T any] struct {
	provider func() (T, error)
	onceLoad *sync.Once
	isLoaded bool
	value    T
	err      error
}

func New[T any](provider func() (T, error)) Loader[T] {
	var empty T
	return &loader[T]{
		provider: provider,
		onceLoad: &sync.Once{},
		isLoaded: false,
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

		l.isLoaded = true
		l.value = value
	})

	return l.value, l.err
}

func (l *loader[T]) IfLoaded(f func(T)) {
	if l.isLoaded {
		f(l.value)
	}
}

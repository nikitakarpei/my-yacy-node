// Package optional holds a value that can be absent, and reads and writes it as JSON, where an
// absent value is null.
package optional

import (
	"encoding/json"
	"fmt"
)

type Value[T any] struct {
	value   T
	present bool
}

func Some[T any](value T) Value[T] {
	return Value[T]{value: value, present: true}
}

func None[T any]() Value[T] {
	return Value[T]{}
}

func (v Value[T]) Get() (T, bool) {
	return v.value, v.present
}

func (v Value[T]) Present() bool {
	return v.present
}

func (v Value[T]) OrElse(fallback T) T {
	if !v.present {
		return fallback
	}

	return v.value
}

func (v Value[T]) MarshalJSON() ([]byte, error) {
	if !v.present {
		return []byte("null"), nil
	}
	data, err := json.Marshal(v.value)
	if err != nil {
		return nil, fmt.Errorf("marshal optional value: %w", err)
	}

	return data, nil
}

func (v *Value[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*v = None[T]()

		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("unmarshal optional value: %w", err)
	}
	*v = Some(value)

	return nil
}

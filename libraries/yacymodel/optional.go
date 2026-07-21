package yacymodel

import (
	"encoding/json"
	"fmt"
)

type Optional[T any] struct {
	value   T
	present bool
}

func Some[T any](value T) Optional[T] {
	return Optional[T]{value: value, present: true}
}

func None[T any]() Optional[T] {
	return Optional[T]{}
}

func (o Optional[T]) Get() (T, bool) {
	return o.value, o.present
}

func (o Optional[T]) Present() bool {
	return o.present
}

func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.present {
		return []byte("null"), nil
	}

	data, err := json.Marshal(o.value)
	if err != nil {
		return nil, fmt.Errorf("marshal optional value: %w", err)
	}

	return data, nil
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = None[T]()

		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("unmarshal optional value: %w", err)
	}
	*o = Some(value)

	return nil
}

package vault

import "fmt"

type storedValue[T any] struct {
	value   T
	encoded []byte
}

type recordCodec[K, V any] struct {
	keys   KeyLayout[K]
	values ValueCodec[V]
}

func (c recordCodec[K, V]) encodedKeyOf(key K) []byte {
	return c.keys.Encode(key).encoded
}

func (c recordCodec[K, V]) keyFrom(encodedKey []byte) (K, error) {
	key, err := c.keys.Decode(encodedKey)
	if err != nil {
		return key, fmt.Errorf("%w: decode key: %w", errBadRecord, err)
	}

	return key, nil
}

func (c recordCodec[K, V]) storedValueOf(val V) (storedValue[V], error) {
	payload, err := c.values.Encode(val)
	if err != nil {
		return storedValue[V]{}, fmt.Errorf("encode value: %w", err)
	}

	return storedValue[V]{value: val, encoded: recordFrom(payload)}, nil
}

func (c recordCodec[K, V]) valueFrom(record []byte) (V, error) {
	var zero V

	payload, err := payloadOf(record)
	if err != nil {
		return zero, err
	}

	val, err := c.values.Decode(payload)
	if err != nil {
		return zero, fmt.Errorf("%w: decode value: %w", errBadRecord, err)
	}

	return val, nil
}

func (c recordCodec[K, V]) storedValueFrom(record []byte) (storedValue[V], error) {
	val, err := c.valueFrom(record)
	if err != nil {
		return storedValue[V]{}, err
	}

	return storedValue[V]{value: val, encoded: record}, nil
}

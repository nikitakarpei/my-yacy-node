package pebblevault

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/cockroachdb/pebble/v2"
)

var errBadAmount = errors.New("bad amount")

var amountMerger = &pebble.Merger{
	Name: "pebblevault.amount_sum",
	Merge: func(_, operand []byte) (pebble.ValueMerger, error) {
		var summed amountSum

		return &summed, summed.add(operand)
	},
}

type amountSum struct {
	total int64
}

func (s *amountSum) add(operand []byte) error {
	change, err := amountFrom(operand)
	s.total += change

	return err
}

func (s *amountSum) MergeNewer(operand []byte) error { return s.add(operand) }

func (s *amountSum) MergeOlder(operand []byte) error { return s.add(operand) }

func (s *amountSum) Finish(bool) ([]byte, io.Closer, error) {
	return encodedAmount(s.total), nil, nil
}

func amountFrom(raw []byte) (int64, error) {
	amount, length := binary.Varint(raw)
	if length <= 0 || length != len(raw) {
		return 0, errBadAmount
	}

	return amount, nil
}

func encodedAmount(amount int64) []byte {
	return binary.AppendVarint(nil, amount)
}

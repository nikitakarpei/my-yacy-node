package pebblevault

import (
	"encoding/binary"
	"errors"
	"io"
)

var errBadAmount = errors.New("bad amount")

type amountSum struct {
	total int64
}

func amountSumFrom(operand []byte) (*amountSum, error) {
	var summed amountSum

	return &summed, summed.add(operand)
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

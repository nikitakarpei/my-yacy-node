package vault

import (
	"encoding/binary"
)

type usedAllowances struct {
	pageOfHost bool
	deferrals  int
	attempts   int
}

type usedAllowancesCodec struct{}

func (usedAllowancesCodec) Encode(record usedAllowances) ([]byte, error) {
	encoded := make([]byte, 0, 1+2*binary.MaxVarintLen64)
	if record.pageOfHost {
		encoded = append(encoded, 1)
	} else {
		encoded = append(encoded, 0)
	}
	encoded = binary.AppendVarint(encoded, int64(record.deferrals))
	return binary.AppendVarint(encoded, int64(record.attempts)), nil
}

func (usedAllowancesCodec) Decode(stored []byte) (usedAllowances, error) {
	if len(stored) == 0 {
		return usedAllowances{}, errLimitMalformed
	}
	deferrals, deferralsBytes := binary.Varint(stored[1:])
	if deferralsBytes <= 0 {
		return usedAllowances{}, errLimitMalformed
	}
	attempts, attemptsBytes := binary.Varint(stored[1+deferralsBytes:])
	if attemptsBytes <= 0 {
		return usedAllowances{}, errLimitMalformed
	}
	return usedAllowances{
		pageOfHost: stored[0] == 1,
		deferrals:  int(deferrals),
		attempts:   int(attempts),
	}, nil
}

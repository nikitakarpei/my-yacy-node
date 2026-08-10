package vault

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var errBadRecord = errors.New("bad record")

const noMetadata = 0

func recordFrom(payload []byte) []byte {
	record := make([]byte, 0, 1+len(payload))
	record = binary.AppendUvarint(record, noMetadata)

	return append(record, payload...)
}

func payloadOf(record []byte) ([]byte, error) {
	metadataLength, headerLength := binary.Uvarint(record)
	if headerLength <= 0 {
		return nil, fmt.Errorf("%w: unreadable metadata length", errBadRecord)
	}
	if metadataLength != noMetadata {
		return nil, fmt.Errorf(
			"%w: %d metadata bytes from a later writer",
			errBadRecord,
			metadataLength,
		)
	}

	return record[headerLength:], nil
}

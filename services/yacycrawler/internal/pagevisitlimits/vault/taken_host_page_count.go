package vault

import (
	"encoding/binary"
)

type takenHostPageCount struct {
	pages int
}

type takenHostPageCountCodec struct{}

func (takenHostPageCountCodec) Encode(record takenHostPageCount) ([]byte, error) {
	return binary.AppendVarint(nil, int64(record.pages)), nil
}

func (takenHostPageCountCodec) Decode(stored []byte) (takenHostPageCount, error) {
	pages, pagesBytes := binary.Varint(stored)
	if pagesBytes != len(stored) {
		return takenHostPageCount{}, errLimitMalformed
	}
	return takenHostPageCount{pages: int(pages)}, nil
}

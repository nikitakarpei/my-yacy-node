package yacymodel

import "strings"

const urlComponentSeparators = " /()-:_.,?!'\""

func URLComponentsOf(address string) uint8 {
	if schemeEnd := strings.Index(address, "//"); schemeEnd > 0 {
		address = address[schemeEnd+len("//"):]
	}
	if !strings.ContainsAny(address, urlComponentSeparators) {
		return 1
	}
	components, emptyPieces, pieceLength := 0, 0, 0
	for i := range len(address) {
		if strings.IndexByte(urlComponentSeparators, address[i]) < 0 {
			pieceLength++

			continue
		}
		if pieceLength == 0 {
			emptyPieces++

			continue
		}
		components += emptyPieces + 1
		emptyPieces, pieceLength = 0, 0
	}
	if pieceLength > 0 {
		components += emptyPieces + 1
	}

	return WrappedCardinalFrom(components)
}

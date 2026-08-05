package searchtest

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

func HashFor(base string) yacymodel.Hash {
	const filler = "AAAAAAAAAAAA"
	padded := base + filler[len(base):]
	if len(base) >= yacymodel.HashLength {
		padded = base[:yacymodel.HashLength]
	}
	hash, err := yacymodel.ParseHash(padded)
	if err != nil {
		panic(err)
	}

	return hash
}

func URLHashFor(url string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(HashFor(url).String())
	if err != nil {
		panic(err)
	}

	return hash
}

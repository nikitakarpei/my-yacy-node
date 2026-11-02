package indextree

import "bytes"

const (
	WordLength = 9
	URLLength  = 9
	KeyLength  = WordLength + URLLength
	keyBits    = KeyLength * 8
	wordBits   = WordLength * 8
)

type (
	Word [WordLength]byte
	URL  [URLLength]byte
	Key  [KeyLength]byte
)

func KeyOf(word Word, url URL) Key {
	var key Key
	copy(key[:WordLength], word[:])
	copy(key[WordLength:], url[:])

	return key
}

func (k Key) Word() Word {
	var word Word
	copy(word[:], k[:WordLength])

	return word
}

func (k Key) URL() URL {
	var url URL
	copy(url[:], k[WordLength:])

	return url
}

func (k Key) bitAt(index int) byte {
	return (k[index/8] >> (7 - uint(index%8))) & 1
}

func (k Key) prefixOfLength(length int) Key {
	var prefix Key
	copy(prefix[:length/8], k[:length/8])
	if length%8 != 0 {
		prefix[length/8] = k[length/8] & (0xff << (8 - uint(length%8)))
	}

	return prefix
}

func (k Key) sharesPrefixWith(other Key, length int) bool {
	return k.prefixOfLength(length) == other.prefixOfLength(length)
}

func firstDifferingBitOf(a, b Key) int {
	for index := range keyBits {
		if a.bitAt(index) != b.bitAt(index) {
			return index
		}
	}

	return keyBits
}

func (k Key) compare(other Key) int {
	return bytes.Compare(k[:], other[:])
}

func wordPrefixOf(word Word) Key {
	return KeyOf(word, URL{})
}

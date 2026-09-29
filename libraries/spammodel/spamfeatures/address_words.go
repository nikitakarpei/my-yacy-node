package spamfeatures

import (
	"strings"
	"unicode"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

func addressTextOf(address canonicalurl.CanonicalURL) string {
	webAddress := address.WebAddress()
	addressParts := webAddress.Hostname() + " " + webAddress.EscapedPath() + " " + webAddress.RawQuery
	return strings.Join(strings.FieldsFunc(strings.ToLower(addressParts), isWordSeparator), " ")
}

func isWordSeparator(character rune) bool {
	return character > unicode.MaxASCII ||
		!unicode.IsLetter(character) && !unicode.IsDigit(character)
}

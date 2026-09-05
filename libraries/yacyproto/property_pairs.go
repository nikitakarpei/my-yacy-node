package yacyproto

import (
	"errors"
	"fmt"
	"strings"
)

var errBadPropertyForm = errors.New("bad property form")

func propertyPairsOfRow(row string) (map[string]string, error) {
	if len(row) < 2 || row[0] != propertyOpen || row[len(row)-1] != propertyClose {
		return nil, fmt.Errorf("%w: missing property form", errBadPropertyForm)
	}

	return parsePropertyPairs(row[1 : len(row)-1])
}

func parsePropertyPairs(body string) (map[string]string, error) {
	props := make(map[string]string)
	for token := range strings.SplitSeq(body, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		pos := strings.IndexByte(token, '=')
		if pos <= 0 {
			return nil, fmt.Errorf("%w: %q", errBadPropertyForm, token)
		}
		props[strings.TrimSpace(token[:pos])] = strings.TrimSpace(token[pos+1:])
	}
	return props, nil
}

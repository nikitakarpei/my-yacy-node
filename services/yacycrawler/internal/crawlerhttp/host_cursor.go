package crawlerhttp

import (
	"encoding/base64"
	"strconv"
)

func hostCursorAt(position int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(position)))
}

func hostPositionOf(cursor string) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, errUnreadableCursor
	}
	position, err := strconv.Atoi(string(decoded))
	if err != nil || position < 0 {
		return 0, errUnreadableCursor
	}
	return position, nil
}

package yacyproto

import "io"

const messageMaxPlainBytes = 4 << 20

type plainTextAllowance struct {
	remainingBytes int
}

func (a *plainTextAllowance) readerWithin(r io.Reader) io.Reader {
	return io.LimitReader(r, int64(a.remainingBytes)+1)
}

func (a *plainTextAllowance) take(plain string) (taken bool) {
	if len(plain) > a.remainingBytes {
		a.remainingBytes = 0

		return false
	}
	a.remainingBytes -= len(plain)

	return true
}

package readingcancel

import (
	"io"
	"time"
)

func IdleCancellingReaderFrom(
	bodyReader io.Reader,
	canceller *Canceller,
	idleTimeout time.Duration,
) io.Reader {
	return idleCancellingReader{
		bodyReader:  bodyReader,
		canceller:   canceller,
		idleTimeout: idleTimeout,
	}
}

type idleCancellingReader struct {
	bodyReader  io.Reader
	canceller   *Canceller
	idleTimeout time.Duration
}

func (r idleCancellingReader) Read(chunk []byte) (int, error) {
	r.canceller.cancelAfter(r.idleTimeout)
	defer r.canceller.Stop()
	//nolint:wrapcheck // io.EOF reaches the caller of a reader unwrapped
	return r.bodyReader.Read(chunk)
}

package proxyintake

import "io"

type clientWriter struct {
	writer      io.Writer
	writeFailed bool
}

func (w *clientWriter) Write(chunk []byte) (int, error) {
	written, err := w.writer.Write(chunk)
	w.writeFailed = err != nil
	return written, err //nolint:wrapcheck // io.Copy hands the cause of the client to the relay
}

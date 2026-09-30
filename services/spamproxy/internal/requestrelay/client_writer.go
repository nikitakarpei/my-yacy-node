package requestrelay

type clientWriter struct {
	responseWriter ResponseWriter
	writeFailed    bool
}

func (w *clientWriter) Write(chunk []byte) (int, error) {
	written, err := w.responseWriter.Write(chunk)
	w.writeFailed = err != nil
	return written, err //nolint:wrapcheck // io.Copy hands the cause of the client to the relay
}

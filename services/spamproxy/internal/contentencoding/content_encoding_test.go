package contentencoding_test

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/contentencoding"
)

var pageBody = []byte("<html><title>Cheap pills</title><p>Buy cheap pills now</p></html>")

func TestARequestAsksOnlyForTheEncodingsTheProxyDecodes(t *testing.T) {
	acceptEncoding := contentencoding.DecodableAcceptEncodingFrom("gzip;q=1.0, br, Deflate, zstd")

	if want := "gzip;q=1.0, Deflate"; acceptEncoding != want {
		t.Fatalf("accept encoding = %q, want %q", acceptEncoding, want)
	}
}

func TestARequestWithNoDecodableEncodingAsksForIdentity(t *testing.T) {
	if acceptEncoding := contentencoding.DecodableAcceptEncodingFrom(
		"br",
	); acceptEncoding != "identity" {
		t.Fatalf("accept encoding = %q, want identity", acceptEncoding)
	}
}

func TestBrotliIsNotDecodable(t *testing.T) {
	if contentencoding.IsDecodable("br") {
		t.Fatal("br is decodable")
	}
}

func TestEachDecodableEncodingGivesTheBody(t *testing.T) {
	encodings := []struct {
		name            string
		contentEncoding string
		encodedBody     []byte
	}{
		{"none", "", pageBody},
		{"identity", "identity", pageBody},
		{"gzip", "gzip", encodedWith(t, gzipWriterOf)},
		{"x-gzip", "x-gzip", encodedWith(t, gzipWriterOf)},
		{"zlib deflate", "deflate", encodedWith(t, zlibWriterOf)},
		{"raw deflate", "deflate", encodedWith(t, rawDeflateWriterOf)},
	}
	for _, encoding := range encodings {
		t.Run(encoding.name, func(t *testing.T) {
			body, decoded := contentencoding.DecodedBodyFrom(
				encoding.encodedBody,
				encoding.contentEncoding,
				1000,
			)
			if !decoded || !bytes.Equal(body, pageBody) {
				t.Fatalf("body = %q, decoded %v", body, decoded)
			}
		})
	}
}

func TestADecodedBodyStopsAtTheByteCeiling(t *testing.T) {
	body, _ := contentencoding.DecodedBodyFrom(
		encodedWith(t, gzipWriterOf), "gzip", 10,
	)

	if !bytes.Equal(body, pageBody[:10]) {
		t.Fatalf("body = %q, want %q", body, pageBody[:10])
	}
}

func TestTheFirstBytesOfAGzipBodyDecode(t *testing.T) {
	encodedBody := encodedWith(t, gzipWriterOf)

	body, decoded := contentencoding.DecodedBodyFrom(
		encodedBody[:len(encodedBody)-12],
		"gzip",
		1000,
	)

	if !decoded || !bytes.HasPrefix(pageBody, body) {
		t.Fatalf("body = %q, decoded %v", body, decoded)
	}
}

func TestABodyThatIsNotGzipDoesNotDecode(t *testing.T) {
	if _, decoded := contentencoding.DecodedBodyFrom([]byte("zipped"), "gzip", 1000); decoded {
		t.Fatal("decoded")
	}
}

func encodedWith(t *testing.T, writerOf func(io.Writer) io.WriteCloser) []byte {
	t.Helper()
	var encodedBody bytes.Buffer
	writer := writerOf(&encodedBody)
	if _, err := writer.Write(pageBody); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return encodedBody.Bytes()
}

func rawDeflateWriterOf(w io.Writer) io.WriteCloser {
	writer, _ := flate.NewWriter(w, flate.DefaultCompression)
	return writer
}

func gzipWriterOf(w io.Writer) io.WriteCloser {
	return gzip.NewWriter(w)
}

func zlibWriterOf(w io.Writer) io.WriteCloser {
	return zlib.NewWriter(w)
}

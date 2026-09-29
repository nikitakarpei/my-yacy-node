// Package contentencoding decodes the content encodings that the proxy can
// assess, and narrows a request to them.
package contentencoding

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"io"
	"strings"
)

const identityEncoding = "identity"

type bodyDecoding func(encodedBody []byte, byteCeiling int) ([]byte, bool)

type decoderOpening func(encodedBodyReader io.Reader) (io.Reader, error)

var bodyDecodings = map[string][]bodyDecoding{
	identityEncoding: {identityBodyOf},
	"":               {identityBodyOf},
	"gzip":           {decodingBy(gzipDecoderOf)},
	"x-gzip":         {decodingBy(gzipDecoderOf)},
	"deflate":        {decodingBy(zlibDecoderOf), decodingBy(rawDeflateDecoderOf)},
}

func IsDecodable(contentEncoding string) bool {
	_, found := bodyDecodings[contentEncoding]
	return found
}

func DecodableAcceptEncodingFrom(acceptEncoding string) string {
	var decodableCodings []string
	for coding := range strings.SplitSeq(acceptEncoding, ",") {
		name, _, _ := strings.Cut(coding, ";")
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" && IsDecodable(name) {
			decodableCodings = append(decodableCodings, strings.TrimSpace(coding))
		}
	}
	if len(decodableCodings) == 0 {
		return identityEncoding
	}
	return strings.Join(decodableCodings, ", ")
}

func DecodedBodyFrom(encodedBody []byte, contentEncoding string, byteCeiling int) ([]byte, bool) {
	for _, decode := range bodyDecodings[contentEncoding] {
		if body, decoded := decode(encodedBody, byteCeiling); decoded {
			return body, true
		}
	}
	return nil, false
}

func identityBodyOf(encodedBody []byte, byteCeiling int) ([]byte, bool) {
	return encodedBody[:min(len(encodedBody), byteCeiling)], true
}

func decodingBy(openDecoder decoderOpening) bodyDecoding {
	return func(encodedBody []byte, byteCeiling int) ([]byte, bool) {
		decoder, err := openDecoder(bytes.NewReader(encodedBody))
		if err != nil {
			return nil, false
		}
		body, err := io.ReadAll(io.LimitReader(decoder, int64(byteCeiling)))
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, false
		}
		return body, true
	}
}

func gzipDecoderOf(encodedBodyReader io.Reader) (io.Reader, error) {
	return gzip.NewReader(encodedBodyReader) //nolint:wrapcheck // the error only means undecodable
}

func zlibDecoderOf(encodedBodyReader io.Reader) (io.Reader, error) {
	return zlib.NewReader(encodedBodyReader) //nolint:wrapcheck // the error only means undecodable
}

func rawDeflateDecoderOf(encodedBodyReader io.Reader) (io.Reader, error) {
	return flate.NewReader(encodedBodyReader), nil
}

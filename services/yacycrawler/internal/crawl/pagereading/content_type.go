package pagereading

import (
	"fmt"
	"mime"
	"strings"
)

const parameterCharset = "charset"

type ContentType struct {
	declared  string
	mediaType string
	charset   string
}

func ContentTypeOf(declared string) (ContentType, error) {
	mediaType, parameters, err := mime.ParseMediaType(declared)
	if err != nil {
		return ContentType{declared: declared, mediaType: leadingSegmentOf(declared)},
			fmt.Errorf("parse content type %q: %w", declared, err)
	}
	return ContentType{
		declared:  declared,
		mediaType: mediaType,
		charset:   parameters[parameterCharset],
	}, nil
}

func leadingSegmentOf(declared string) string {
	return strings.ToLower(strings.TrimSpace(strings.SplitN(declared, ";", 2)[0]))
}

func (contentType ContentType) String() string {
	return contentType.declared
}

func (contentType ContentType) MediaType() string {
	return contentType.mediaType
}

func (contentType ContentType) Charset() string {
	return contentType.charset
}

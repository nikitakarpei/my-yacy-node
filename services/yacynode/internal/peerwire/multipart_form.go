package peerwire

import (
	"bytes"
	"maps"
	"mime/multipart"
	"net/url"
	"slices"
)

type multipartForm struct {
	content     []byte
	contentType string
}

func multipartFormFrom(form url.Values) multipartForm {
	var content bytes.Buffer
	writer := multipart.NewWriter(&content)
	for _, key := range slices.Sorted(maps.Keys(form)) {
		for _, value := range form[key] {
			_ = writer.WriteField(key, value)
		}
	}
	_ = writer.Close()

	return multipartForm{content: content.Bytes(), contentType: writer.FormDataContentType()}
}

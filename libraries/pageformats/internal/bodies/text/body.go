// Package text holds a derived body that is already text, such as plain text or
// markdown, and gives its bytes as they are.
package text

type Body struct {
	bytes []byte
}

func New(bytes []byte) Body {
	return Body{bytes: bytes}
}

func (body Body) Bytes() ([]byte, error) {
	return body.bytes, nil
}

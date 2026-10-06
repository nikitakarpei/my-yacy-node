package documentextraction

type Document struct {
	Title         string
	Body          Body
	Format        Format
	Language      string
	LocalLinks    int
	ExternalLinks int
}

type Body interface {
	Bytes() ([]byte, error)
}

package documentextraction

type Document struct {
	Title                string
	Body                 []byte
	Format               Format
	Language             string
	LocalLinks           int
	ExternalLinks        int
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
}

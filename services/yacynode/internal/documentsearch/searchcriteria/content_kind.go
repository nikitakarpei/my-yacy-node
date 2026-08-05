package searchcriteria

type ContentKind int

const (
	AnyContent ContentKind = iota
	ImageContent
	AudioContent
	VideoContent
	ApplicationContent
)

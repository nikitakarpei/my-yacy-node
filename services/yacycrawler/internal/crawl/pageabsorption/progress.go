package pageabsorption

const (
	DisposalOversized            = "oversized"
	DisposalUnsupportedMediaType = "unsupported-media-type"
	DisposalContainerOverflow    = "container-overflow"
	DisposalUnextractable        = "unextractable"
	DisposalNoIndex              = "noindex"
	DisposalUnrepresentable      = "unrepresentable"
)

type Progress interface {
	PageDisposed(reason string)
	PagePublished(representation string)
	PublicationWaited()
}

package requestrelay

type ReplyKind string

const (
	AssessedReply ReplyKind = "assessed"
	SkippedReply  ReplyKind = "skipped"
	RefusedReply  ReplyKind = "refused"
	FailedReply   ReplyKind = "failed"
)

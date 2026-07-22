package ordertraversal

const (
	DisposalBudgetTruncated = "budget-truncated"

	RefusalDefer = "defer"
)

type Progress interface {
	PageFetched()
	PageDisposed(reason string)
	RefusalHonored(demand string)
	BudgetExhausted()
}

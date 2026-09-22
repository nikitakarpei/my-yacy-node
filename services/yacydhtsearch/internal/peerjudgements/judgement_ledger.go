package peerjudgements

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type JudgementLedger interface {
	JudgementOf(
		ctx context.Context,
		peer yacymodel.Hash,
		question Question,
	) yacymodel.Optional[RecordedJudgement]
	HoldJudgement(ctx context.Context, judgement RecordedJudgement)
}

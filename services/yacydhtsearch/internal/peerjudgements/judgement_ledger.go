package peerjudgements

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type JudgementLedger interface {
	JudgementOf(
		ctx context.Context,
		form Form,
		peer yacymodel.Hash,
	) yacymodel.Optional[RecordedJudgement]
	HoldJudgement(ctx context.Context, judgement RecordedJudgement)
}

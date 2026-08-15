package urlmetadatacourier_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/urlmetadatacourier"
)

type scriptedURLMetadataCourier struct {
	receipts []urlmetadatacourier.Receipt
	calls    [][]yacymodel.URLMetadata
}

func (c *scriptedURLMetadataCourier) Deliver(
	_ context.Context,
	_ string,
	_ yacymodel.Hash,
	metadata []yacymodel.URLMetadata,
) urlmetadatacourier.Receipt {
	call := len(c.calls)
	c.calls = append(c.calls, metadata)

	if call < len(c.receipts) {
		return c.receipts[call]
	}

	return urlmetadatacourier.Receipt{Outcome: urlmetadatacourier.Accepted}
}

func metadataRows(n int) []yacymodel.URLMetadata {
	rows := make([]yacymodel.URLMetadata, n)
	for i := range rows {
		rows[i] = yacymodel.URLMetadata{Address: "http://example.com/" + string(rune('a'+i))}
	}

	return rows
}

func TestBoundedDeliverEmptyInputSendsNoBatches(t *testing.T) {
	inner := &scriptedURLMetadataCourier{}
	courier := urlmetadatacourier.NewBounded(inner, 2)

	receipt := courier.Deliver(context.Background(), "peer:8090", courierHash("peer"), nil)

	if receipt.Outcome != urlmetadatacourier.Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Accepted)
	}
	if len(inner.calls) != 0 {
		t.Fatalf("inner calls = %d, want 0", len(inner.calls))
	}
}

func TestBoundedDeliverInputSmallerThanBatchSizeSendsOneBatch(t *testing.T) {
	inner := &scriptedURLMetadataCourier{
		receipts: []urlmetadatacourier.Receipt{{Outcome: urlmetadatacourier.Accepted}},
	}
	courier := urlmetadatacourier.NewBounded(inner, 50)

	receipt := courier.Deliver(
		context.Background(),
		"peer:8090",
		courierHash("peer"),
		metadataRows(10),
	)

	if receipt.Outcome != urlmetadatacourier.Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Accepted)
	}
	if len(inner.calls) != 1 || len(inner.calls[0]) != 10 {
		t.Fatalf("inner calls = %v, want one call of 10 rows", inner.calls)
	}
}

func TestBoundedDeliverSplitsIntoBatches(t *testing.T) {
	inner := &scriptedURLMetadataCourier{
		receipts: []urlmetadatacourier.Receipt{
			{Outcome: urlmetadatacourier.Accepted},
			{Outcome: urlmetadatacourier.Accepted},
			{Outcome: urlmetadatacourier.Accepted},
		},
	}
	courier := urlmetadatacourier.NewBounded(inner, 2)

	receipt := courier.Deliver(
		context.Background(),
		"peer:8090",
		courierHash("peer"),
		metadataRows(5),
	)

	if receipt.Outcome != urlmetadatacourier.Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Accepted)
	}

	wantSizes := []int{2, 2, 1}
	if len(inner.calls) != len(wantSizes) {
		t.Fatalf("inner calls = %d, want %d", len(inner.calls), len(wantSizes))
	}
	for i, want := range wantSizes {
		if len(inner.calls[i]) != want {
			t.Fatalf("call %d size = %d, want %d", i, len(inner.calls[i]), want)
		}
	}
}

func TestBoundedDeliverAccumulatesRejectedURLsAcrossBatches(t *testing.T) {
	first := urlHash("u1")
	second := urlHash("u2")
	inner := &scriptedURLMetadataCourier{
		receipts: []urlmetadatacourier.Receipt{
			{Outcome: urlmetadatacourier.Accepted, URLsRejected: []yacymodel.URLHash{first}},
			{Outcome: urlmetadatacourier.Accepted, URLsRejected: []yacymodel.URLHash{second}},
		},
	}
	courier := urlmetadatacourier.NewBounded(inner, 2)

	receipt := courier.Deliver(
		context.Background(),
		"peer:8090",
		courierHash("peer"),
		metadataRows(4),
	)

	if receipt.Outcome != urlmetadatacourier.Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Accepted)
	}
	if len(receipt.URLsRejected) != 2 || receipt.URLsRejected[0] != first ||
		receipt.URLsRejected[1] != second {
		t.Fatalf("URLsRejected = %v, want [%v %v]", receipt.URLsRejected, first, second)
	}
}

func TestBoundedDeliverStopsAfterFirstFailedBatch(t *testing.T) {
	outcomes := []urlmetadatacourier.Outcome{
		urlmetadatacourier.Deferred,
		urlmetadatacourier.Refused,
		urlmetadatacourier.Unreachable,
	}

	for _, outcome := range outcomes {
		t.Run(string(outcome), func(t *testing.T) {
			inner := &scriptedURLMetadataCourier{
				receipts: []urlmetadatacourier.Receipt{
					{Outcome: urlmetadatacourier.Accepted},
					{Outcome: outcome},
				},
			}
			courier := urlmetadatacourier.NewBounded(inner, 2)

			receipt := courier.Deliver(
				context.Background(),
				"peer:8090",
				courierHash("peer"),
				metadataRows(6),
			)

			if receipt.Outcome != outcome {
				t.Fatalf("outcome = %q, want %q", receipt.Outcome, outcome)
			}
			if receipt.URLsRejected != nil {
				t.Fatalf("URLsRejected = %v, want nil", receipt.URLsRejected)
			}
			if len(inner.calls) != 2 {
				t.Fatalf(
					"inner calls = %d, want 2 (third batch must not be sent)",
					len(inner.calls),
				)
			}
		})
	}
}

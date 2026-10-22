package vault_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestExpiryObserversTellEveryObserverEachReport(t *testing.T) {
	first, second := &removalReports{}, &removalReports{}
	observers := vault.ExpiryObservers{first, second}
	ctx := context.Background()

	observers.ExpiryPassBegan(ctx, "words", testEpoch)
	observers.ExpiryPassFinished(ctx, "words", testEpoch)
	observers.ExpiredRecordsForgotten(ctx, "words", 3)
	observers.ExpiredRecordsNotForgotten(ctx, "words", errUnreadableMeaning)

	for _, reports := range []*removalReports{first, second} {
		if reports.passesBegan != 1 || reports.passesFinished != 1 || reports.forgotten != 3 ||
			len(reports.notForgotten) != 1 ||
			!errors.Is(reports.notForgotten[0], errUnreadableMeaning) {
			t.Fatalf("reports = %+v, want each report once", reports)
		}
	}
}

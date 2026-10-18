package vault

import (
	"context"
	"log/slog"
)

type MemberSetObserver interface {
	ObserveMemberAdded(bucket Name, form MemberForm)
	ObserveUndecodableMembers(bucket Name, amount int)
}

type silentMemberSetObserver struct{}

func (silentMemberSetObserver) ObserveMemberAdded(Name, MemberForm) {}

func (silentMemberSetObserver) ObserveUndecodableMembers(Name, int) {}

func memberSetObserverOrSilent(observer MemberSetObserver) MemberSetObserver {
	if observer == nil {
		return silentMemberSetObserver{}
	}

	return observer
}

func reportUndecodableMembers(observer MemberSetObserver, bucket Name, amount int, cause error) {
	observer.ObserveUndecodableMembers(bucket, amount)
	slog.ErrorContext(
		context.Background(),
		"vault member set holds members it cannot decode",
		slog.String("bucket", string(bucket)),
		slog.Int("members", amount),
		slog.Any("error", cause),
	)
}

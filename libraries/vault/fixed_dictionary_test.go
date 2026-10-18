package vault_test

import (
	"context"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func formsOfMentions(
	t *testing.T,
	engine *doubleEngine,
	valuesByID []string,
	words ...string,
) []vault.MemberForm {
	t.Helper()

	opened := openPageWords(t, engine, valuesByID)
	opened.observed.addedForms = nil
	opened.mention(t, "p9", words...)

	return opened.observed.addedForms
}

func assertForms(t *testing.T, got []vault.MemberForm, want ...vault.MemberForm) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("forms = %v, want %v", got, want)
	}
}

func TestADictionaryRegisteredWithoutValuesUsesTheStoredCopy(t *testing.T) {
	engine := newDoubleEngine()
	openPageWords(t, engine, []string{"the", "a"})

	got := formsOfMentions(t, engine, nil, "a", "zebra")

	assertForms(t, got, vault.MemberFormID, vault.MemberFormRaw)
}

func TestNewValuesReplaceTheStoredCopy(t *testing.T) {
	engine := newDoubleEngine()
	openPageWords(t, engine, []string{"the"})
	openPageWords(t, engine, []string{"zebra"})

	got := formsOfMentions(t, engine, nil, "the", "zebra")

	assertForms(t, got, vault.MemberFormRaw, vault.MemberFormID)
}

func TestWithoutValuesOrAStoredCopyEveryMemberIsRaw(t *testing.T) {
	got := formsOfMentions(t, newDoubleEngine(), nil, "the", "zebra")

	assertForms(t, got, vault.MemberFormRaw, vault.MemberFormRaw)
}

func TestASecondRegistrationOfADictionaryIsRefused(t *testing.T) {
	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	ctx := context.Background()
	if _, err := v.RegisterFixedDictionary(
		ctx,
		pageWordsDictionary,
		vault.TextKeyPart,
		nil,
	); err != nil {
		t.Fatalf("RegisterFixedDictionary: %v", err)
	}

	if _, err := v.RegisterFixedDictionary(
		ctx,
		pageWordsDictionary,
		vault.TextKeyPart,
		nil,
	); err == nil {
		t.Fatal("second RegisterFixedDictionary succeeded, want refusal")
	}
}

func TestValuesThatRepeatAValueAreRefused(t *testing.T) {
	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}

	if _, err := v.RegisterFixedDictionary(
		context.Background(),
		pageWordsDictionary,
		vault.TextKeyPart,
		[]string{"the", "a", "the"},
	); err == nil {
		t.Fatal("RegisterFixedDictionary with a repeated value succeeded, want refusal")
	}
}

func TestRegisteringADictionaryInAClosedVaultIsRefused(t *testing.T) {
	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	if err := v.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := v.RegisterFixedDictionary(
		context.Background(),
		pageWordsDictionary,
		vault.TextKeyPart,
		nil,
	); err == nil {
		t.Fatal("RegisterFixedDictionary on a closed vault succeeded, want refusal")
	}
}

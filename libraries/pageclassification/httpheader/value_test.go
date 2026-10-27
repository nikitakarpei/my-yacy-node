package httpheader_test

import (
	"maps"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification/httpheader"
)

func TestEachCategoryIsAMemberWithItsVerdictScoreThresholdAndModel(t *testing.T) {
	t.Parallel()

	classification := pageclassification.Classification{
		pageclassification.Spam: {Score: 0.93549, Threshold: 0.8, ModelVersion: "2026-09"},
		"adult":                 {Score: 0.2, Threshold: 0.75, ModelVersion: "2026-11"},
	}

	if value, want := httpheader.ValueOf(classification),
		`spam;score=0.935;threshold=0.800;model="2026-09", `+
			`adult=?0;score=0.200;threshold=0.750;model="2026-11"`; value != want {
		t.Fatalf("value = %s, want %s", value, want)
	}
}

func TestTheMembersComeInTheOrderOfTheirScoresTheHighestFirst(t *testing.T) {
	t.Parallel()

	classification := pageclassification.Classification{
		"medical":  {Score: 0.4, Threshold: 0.5},
		"violence": {Score: 0.9, Threshold: 0.5},
		"adult":    {Score: 0.4, Threshold: 0.5},
		"spam":     {Score: 0.1, Threshold: 0.5},
	}

	if value, want := httpheader.ValueOf(classification),
		`violence;score=0.900;threshold=0.500;model="", `+
			`adult=?0;score=0.400;threshold=0.500;model="", `+
			`medical=?0;score=0.400;threshold=0.500;model="", `+
			`spam=?0;score=0.100;threshold=0.500;model=""`; value != want {
		t.Fatalf("value = %s, want %s", value, want)
	}
}

func TestEachMemberGivesTheVerdictOfItsCategory(t *testing.T) {
	t.Parallel()

	for classificationValue, wantedVerdicts := range map[string]pageclassification.Verdicts{
		`spam;score=0.935;threshold=0.8;model="2026-09"`: {"spam": pageclassification.Positive},
		`spam=?1, adult=?0;score=0.1`: {
			"spam":  pageclassification.Positive,
			"adult": pageclassification.Negative,
		},
		`adult=?0;model="a,b;c=?1", spam`: {
			"adult": pageclassification.Negative,
			"spam":  pageclassification.Positive,
		},
		`spam;model="say \"x, y\"", adult`: {
			"spam":  pageclassification.Positive,
			"adult": pageclassification.Positive,
		},
		"spam=?0, spam=?1": {"spam": pageclassification.Positive},
	} {
		if verdicts := httpheader.VerdictsFrom(classificationValue); !maps.Equal(
			verdicts, wantedVerdicts,
		) {
			t.Errorf("%q yields %v, want %v", classificationValue, verdicts, wantedVerdicts)
		}
	}
}

func TestAMemberWithoutABooleanOrACategoryKeyGivesNoVerdict(t *testing.T) {
	t.Parallel()

	for _, classificationValue := range []string{
		"",
		`spam="yes"`,
		"spam=1",
		"spam=(?1)",
		"Spam=?1",
		"spam =?1",
		"*spam=?1",
	} {
		if verdicts := httpheader.VerdictsFrom(classificationValue); len(verdicts) != 0 {
			t.Errorf("%q yields %v, want no verdict", classificationValue, verdicts)
		}
	}
}

func TestTheVerdictsReadFromAValueAreTheVerdictsWritten(t *testing.T) {
	t.Parallel()

	classification := pageclassification.Classification{
		pageclassification.Spam: {Score: 0.9, Threshold: 0.8, ModelVersion: "2026-09"},
		"adult":                 {Score: 0.1, Threshold: 0.8, ModelVersion: `"quoted", v2`},
	}

	if verdicts := httpheader.VerdictsFrom(httpheader.ValueOf(classification)); !maps.Equal(
		verdicts,
		pageclassification.Verdicts{
			pageclassification.Spam: pageclassification.Positive,
			"adult":                 pageclassification.Negative,
		},
	) {
		t.Fatalf("verdicts = %v", verdicts)
	}
}

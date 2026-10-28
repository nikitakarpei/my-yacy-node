package consolepages

import (
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const recrawlGraceDay = hoursPerDay * time.Hour

var (
	defaultRecrawlGrace = recrawlGraceDay
	presetRecrawlGraces = []time.Duration{
		time.Hour,
		6 * time.Hour,
		recrawlGraceDay,
		7 * recrawlGraceDay,
		yacycrawlcontract.MaxRecrawlGrace.Length(),
	}
)

type recrawlGraceOption struct {
	Value string
	shownRecrawlGrace
}

func recrawlGraceOptionsWith(chosenRecrawlGrace string) []recrawlGraceOption {
	lengths := presetRecrawlGraces
	chosen, problem := recrawlGraceFrom(chosenRecrawlGrace)
	if problem.Key == "" && !slices.Contains(lengths, chosen.Length()) {
		lengths = append(slices.Clone(lengths), chosen.Length())
		slices.Sort(lengths)
	}
	options := make([]recrawlGraceOption, 0, len(lengths))
	for _, length := range lengths {
		options = append(options, recrawlGraceOption{
			Value:             length.String(),
			shownRecrawlGrace: shownRecrawlGraceOf(length),
		})
	}
	return options
}

func recrawlGraceFrom(recrawlGrace string) (yacycrawlcontract.RecrawlGrace, formProblem) {
	unreadable := formProblem{Key: "new-crawl.recrawl-grace-unreadable", Value: recrawlGrace}
	length, err := time.ParseDuration(recrawlGrace)
	if err != nil {
		return yacycrawlcontract.NoRecrawlGrace, unreadable
	}
	parsed, err := yacycrawlcontract.RecrawlGraceOf(length)
	if err != nil || parsed == yacycrawlcontract.NoRecrawlGrace {
		return yacycrawlcontract.NoRecrawlGrace, unreadable
	}
	return parsed, formProblem{}
}

type recrawlGraceUnit struct {
	length time.Duration
	one    string
	many   string
}

var recrawlGraceUnits = []recrawlGraceUnit{
	{length: recrawlGraceDay, one: "day", many: "days"},
	{length: time.Hour, one: "hour", many: "hours"},
	{length: time.Minute, one: "minute", many: "minutes"},
}

type shownRecrawlGrace struct {
	Amount int64
	Unit   string
}

func shownRecrawlGraceOf(length time.Duration) shownRecrawlGrace {
	for _, unit := range recrawlGraceUnits {
		if length%unit.length == 0 {
			return unit.shownGraceOf(int64(length / unit.length))
		}
	}
	shortestUnit := recrawlGraceUnits[len(recrawlGraceUnits)-1]
	return shortestUnit.shownGraceOf(
		int64((length + shortestUnit.length - 1) / shortestUnit.length),
	)
}

func (unit recrawlGraceUnit) shownGraceOf(amount int64) shownRecrawlGrace {
	if amount == 1 {
		return shownRecrawlGrace{Amount: amount, Unit: unit.one}
	}
	return shownRecrawlGrace{Amount: amount, Unit: unit.many}
}

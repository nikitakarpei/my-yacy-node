package consolepages

import (
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

var (
	defaultRecrawlGrace = hoursPerDay * time.Hour
	presetRecrawlGraces = []time.Duration{
		time.Hour,
		6 * time.Hour,
		hoursPerDay * time.Hour,
		7 * hoursPerDay * time.Hour,
		yacycrawlcontract.MaxRecrawlGrace,
	}
)

type recrawlGraceOption struct {
	Value   string
	Written writtenRecrawlGrace
}

func recrawlGraceOptionsWith(chosenRecrawlGrace string) []recrawlGraceOption {
	recrawlGraces := presetRecrawlGraces
	chosen, problem := recrawlGraceFrom(chosenRecrawlGrace)
	if problem.Key == "" && !slices.Contains(recrawlGraces, chosen) {
		recrawlGraces = append(slices.Clone(recrawlGraces), chosen)
		slices.Sort(recrawlGraces)
	}
	options := make([]recrawlGraceOption, 0, len(recrawlGraces))
	for _, recrawlGrace := range recrawlGraces {
		options = append(options, recrawlGraceOption{
			Value:   recrawlGrace.String(),
			Written: writtenRecrawlGraceOf(recrawlGrace),
		})
	}
	return options
}

func recrawlGraceFrom(recrawlGrace string) (time.Duration, formProblem) {
	parsed, err := time.ParseDuration(recrawlGrace)
	if err != nil || parsed <= yacycrawlcontract.NoRecrawlGrace ||
		parsed > yacycrawlcontract.MaxRecrawlGrace {
		return 0, formProblem{Key: "new-crawl.recrawl-grace-unreadable", Value: recrawlGrace}
	}
	return parsed, formProblem{}
}

type recrawlGraceUnit struct {
	name   string
	length time.Duration
}

var (
	recrawlGraceMinute = recrawlGraceUnit{name: "minute", length: time.Minute}
	recrawlGraceUnits  = []recrawlGraceUnit{
		{name: "day", length: hoursPerDay * time.Hour},
		{name: "hour", length: time.Hour},
		recrawlGraceMinute,
	}
)

type writtenRecrawlGrace struct {
	Amount int64
	Unit   string
}

func writtenRecrawlGraceOf(recrawlGrace time.Duration) writtenRecrawlGrace {
	for _, unit := range recrawlGraceUnits {
		if recrawlGrace%unit.length == 0 {
			return unit.writtenGraceOf(int64(recrawlGrace / unit.length))
		}
	}
	return recrawlGraceMinute.writtenGraceOf(int64((recrawlGrace + time.Minute - 1) / time.Minute))
}

func (unit recrawlGraceUnit) writtenGraceOf(amount int64) writtenRecrawlGrace {
	if amount == 1 {
		return writtenRecrawlGrace{Amount: amount, Unit: unit.name}
	}
	return writtenRecrawlGrace{Amount: amount, Unit: unit.name + "s"}
}

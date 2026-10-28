package consolepages

import (
	"strconv"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type recrawlGraceUnit struct {
	Name   string
	length time.Duration
}

var (
	recrawlGraceMinutes = recrawlGraceUnit{Name: "minutes", length: time.Minute}
	recrawlGraceHours   = recrawlGraceUnit{Name: "hours", length: time.Hour}
	recrawlGraceDays    = recrawlGraceUnit{Name: "days", length: hoursPerDay * time.Hour}
	recrawlGraceUnits   = []recrawlGraceUnit{
		recrawlGraceMinutes,
		recrawlGraceHours,
		recrawlGraceDays,
	}
)

type writtenRecrawlGrace struct {
	Amount string
	Unit   string
}

func writtenRecrawlGraceOf(recrawlGrace time.Duration) writtenRecrawlGrace {
	if recrawlGrace == yacycrawlcontract.NoRecrawlGrace {
		return writtenRecrawlGrace{Unit: recrawlGraceDays.Name}
	}
	for _, unit := range []recrawlGraceUnit{recrawlGraceDays, recrawlGraceHours} {
		if recrawlGrace%unit.length == 0 {
			return writtenRecrawlGrace{
				Amount: strconv.FormatInt(int64(recrawlGrace/unit.length), 10),
				Unit:   unit.Name,
			}
		}
	}
	startedMinutes := (recrawlGrace + time.Minute - 1) / time.Minute
	return writtenRecrawlGrace{
		Amount: strconv.FormatInt(int64(startedMinutes), 10),
		Unit:   recrawlGraceMinutes.Name,
	}
}

func recrawlGraceFrom(written writtenRecrawlGrace) (time.Duration, formProblem) {
	if written.Amount == "" {
		return yacycrawlcontract.NoRecrawlGrace, formProblem{}
	}
	amount, err := strconv.ParseInt(written.Amount, 10, 64)
	if err != nil || amount < 0 {
		return 0, formProblem{Key: "new-crawl.recrawl-grace-unreadable", Value: written.Amount}
	}
	unit, known := recrawlGraceUnitNamed(written.Unit)
	if !known {
		return 0, formProblem{Key: "new-crawl.recrawl-grace-unit-unknown", Value: written.Unit}
	}
	if amount > int64(yacycrawlcontract.MaxRecrawlGrace/unit.length) {
		return 0, formProblem{
			Key: "new-crawl.recrawl-grace-past-max",
			Value: strconv.FormatInt(
				int64(yacycrawlcontract.MaxRecrawlGrace/recrawlGraceDays.length),
				10,
			),
		}
	}
	return time.Duration(amount) * unit.length, formProblem{}
}

func recrawlGraceUnitNamed(name string) (recrawlGraceUnit, bool) {
	for _, unit := range recrawlGraceUnits {
		if unit.Name == name {
			return unit, true
		}
	}
	return recrawlGraceUnit{}, false
}

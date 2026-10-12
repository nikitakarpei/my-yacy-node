package robotsexclusion

import "time"

type Availability int

const (
	Undetermined Availability = iota
	Stated
	Absent
	Unreachable
)

type Rules struct {
	Availability Availability
	RobotsTxt    string
	Lifetime     time.Duration
}

func (rules Rules) GroupFor(productToken ProductToken) Group {
	return Group{rules: rules, productToken: productToken}
}

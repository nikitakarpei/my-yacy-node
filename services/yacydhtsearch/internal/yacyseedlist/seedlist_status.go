package yacyseedlist

import "fmt"

type statusError int

func (e statusError) Error() string {
	return fmt.Sprintf("seedlist answered with status %d", int(e))
}

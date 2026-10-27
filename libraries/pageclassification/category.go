package pageclassification

import (
	"fmt"
	"regexp"
)

type Category string

const Spam Category = "spam"

var categoryName = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

func CategoryFrom(name string) (Category, error) {
	if !categoryName.MatchString(name) {
		return "", fmt.Errorf("category %q: want a name that matches %s", name, categoryName)
	}
	return Category(name), nil
}

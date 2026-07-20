package yacyproto

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var errBadSoftwareVersion = errors.New("bad software version")

// softwareVersionWirePattern splits YaCy's packed version double into the
// release (major.minor, up to three minor digits) and the appended revision.
var softwareVersionWirePattern = regexp.MustCompile(`\A(\d+\.\d{1,3})(\d{0,5})\z`)

// softwareVersionWireCodec translates between the software version domain type
// and YaCy's packed MAJOR.MINOR{padded}REVISION double.
type softwareVersionWireCodec struct{}

func (softwareVersionWireCodec) decode(text string) (yacymodel.SoftwareVersion, error) {
	match := softwareVersionWirePattern.FindStringSubmatch(text)
	if match == nil {
		return yacymodel.SoftwareVersion{}, fmt.Errorf("%w: %q", errBadSoftwareVersion, text)
	}
	release, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return yacymodel.SoftwareVersion{}, fmt.Errorf("%w: %w", errBadSoftwareVersion, err)
	}
	revision := 0
	if match[2] != "" {
		revision, err = strconv.Atoi(match[2])
		if err != nil {
			return yacymodel.SoftwareVersion{}, fmt.Errorf("%w: %w", errBadSoftwareVersion, err)
		}
	}
	return yacymodel.SoftwareVersion{Release: release, Revision: revision}, nil
}

func (softwareVersionWireCodec) encode(v yacymodel.SoftwareVersion) string {
	return fmt.Sprintf("%.3f%d", v.Release, v.Revision)
}

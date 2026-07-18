package yacymodel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
)

const (
	ColURLHash           = "h"
	ColLastModified      = "a"
	ColFreshUntil        = "s"
	ColTitleWordCount    = "u"
	ColTextWordCount     = "w"
	ColPhraseCount       = "p"
	ColDocType           = "d"
	ColLanguage          = "l"
	ColLocalLinkCount    = "x"
	ColExternalLinkCount = "y"
	ColURLLength         = "m"
	ColURLComponentCount = "n"
	ColWordType          = "g"
	ColFlags             = "z"
	ColHitCount          = "c"
	ColTextPosition      = "t"
	ColPhraseRelativePos = "r"
	ColPhrasePosition    = "o"
	// ColWordDistance is a query-time term-distance signal, always 0 in a stored posting; the node derives it from ColTextPosition and never reads this column.
	ColWordDistance = "i"
	ColReserve      = "k"
	propertyOpen    = '{'
	propertyClose   = '}'
)

const RWIFlagBitCount = 32

const (
	RWIFlagHasImage = 20
	RWIFlagHasAudio = 21
	RWIFlagHasVideo = 22
	RWIFlagHasApp   = 23
)

const (
	DocTypeImage = 'i'
	DocTypeAudio = 'a'
	DocTypeMovie = 'm'
)

const (
	rwiByteFlagLength = 4
	langLength        = 2
)

var ErrBadRWIPosting = errors.New("bad rwi posting")

var errInvalidRWIProperty = errors.New("invalid rwi property")

var rwiCardinalWidths = map[string]int{
	ColLastModified:      2,
	ColFreshUntil:        2,
	ColTitleWordCount:    1,
	ColTextWordCount:     2,
	ColPhraseCount:       2,
	ColLocalLinkCount:    1,
	ColExternalLinkCount: 1,
	ColURLLength:         1,
	ColURLComponentCount: 1,
	ColHitCount:          1,
	ColTextPosition:      2,
	ColPhraseRelativePos: 1,
	ColPhrasePosition:    1,
	ColWordDistance:      1,
	ColReserve:           1,
}

// RWIPostingWireForm is the reverse-word-index posting in its property-form
// wire representation: a flat, protocol-defined column map.
type RWIPostingWireForm struct {
	WordHash   Hash
	Properties map[string]string
}

type RWIPostingID struct {
	WordHash Hash
	URLHash  Hash
}

type WordPostings struct {
	WordHash Hash
	Postings []RWIPostingWireForm
}

func (e RWIPostingWireForm) URLHash() (URLHash, error) {
	return ParseURLHash(e.Properties[ColURLHash])
}

func (e RWIPostingWireForm) DocType() (byte, bool) {
	value, err := e.ByteValue(ColDocType)
	if err != nil {
		slog.WarnContext(context.Background(), "rwi doctype discarded", slog.Any("error", err))
		return 0, false
	}
	return value, true
}

func (e RWIPostingWireForm) AppearanceFlags() (Bitfield, error) {
	value, ok := e.Properties[ColFlags]
	if !ok {
		return nil, nil
	}
	return DecodeBitfield(value)
}

func (e RWIPostingWireForm) Cardinal(key string) (uint64, error) {
	value := e.Properties[key]
	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse rwi cardinal %s: %w", key, err)
	}
	return n, nil
}

func (e RWIPostingWireForm) ByteValue(key string) (byte, error) {
	value := e.Properties[key]
	n, err := strconv.ParseUint(value, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("parse rwi byte %s: %w", key, err)
	}
	return byte(n), nil
}

func ParseRWIPosting(line string) (RWIPostingWireForm, error) {
	open := strings.IndexByte(line, propertyOpen)
	if open < 0 || !strings.HasSuffix(line, string(propertyClose)) {
		return RWIPostingWireForm{}, fmt.Errorf("%w: missing property form", ErrBadRWIPosting)
	}
	wordHash, err := ParseHash(line[:open])
	if err != nil {
		return RWIPostingWireForm{}, fmt.Errorf("%w: word hash: %w", ErrBadRWIPosting, err)
	}
	props, err := parsePropertyPairs(line[open+1 : len(line)-1])
	if err != nil {
		return RWIPostingWireForm{}, fmt.Errorf("%w: %w", ErrBadRWIPosting, err)
	}
	props, err = normalizeRWIProperties(props)
	if err != nil {
		return RWIPostingWireForm{}, fmt.Errorf("%w: %w", ErrBadRWIPosting, err)
	}
	if err := validateRWIProperties(props); err != nil {
		return RWIPostingWireForm{}, fmt.Errorf("%w: %w", ErrBadRWIPosting, err)
	}
	return RWIPostingWireForm{WordHash: wordHash, Properties: props}, nil
}

func (e RWIPostingWireForm) String() string {
	keys := make([]string, 0, len(e.Properties))
	for k := range e.Properties {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	b.WriteString(string(e.WordHash))
	b.WriteByte(propertyOpen)
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(e.Properties[k])
	}
	b.WriteByte(propertyClose)
	return b.String()
}

func FormatRWICardinal(value uint64) string {
	return strconv.FormatUint(value, 10)
}

func validateRWIProperties(props map[string]string) error {
	for key, value := range props {
		if err := validateRWIProperty(key, value); err != nil {
			return err
		}
	}
	return nil
}

func validateRWIProperty(key, value string) error {
	switch key {
	case ColURLHash:
		if _, err := ParseHash(value); err != nil {
			return fmt.Errorf("%w %s: %w", errInvalidRWIProperty, key, err)
		}
	case ColLanguage:
		if len(value) != langLength {
			return fmt.Errorf(
				"%w %s: length %d, want %d",
				errInvalidRWIProperty,
				key,
				len(value),
				langLength,
			)
		}
	case ColFlags:
		return validateOptionalEncoded(key, value)
	case ColDocType, ColWordType:
		if _, err := strconv.ParseUint(value, 10, 8); err != nil {
			return fmt.Errorf("%w %s: %w", errInvalidRWIProperty, key, err)
		}
	default:
		if _, ok := rwiCardinalWidths[key]; ok {
			if _, err := strconv.ParseUint(value, 10, 64); err != nil {
				return fmt.Errorf("%w %s: %w", errInvalidRWIProperty, key, err)
			}
		}
	}
	return nil
}

func validateOptionalEncoded(key, value string) error {
	if _, err := Decode(value); err != nil {
		return fmt.Errorf("%w %s: %w", errInvalidRWIProperty, key, err)
	}

	return nil
}

type rwiDecimalPrefix struct {
	magnitude uint64
	negative  bool
}

func normalizeRWIProperties(props map[string]string) (map[string]string, error) {
	out := make(map[string]string, len(props))
	for key, value := range props {
		normalized, err := normalizeRWIProperty(key, value)
		if err != nil {
			return nil, err
		}
		out[key] = normalized
	}
	return out, nil
}

func normalizeRWIProperty(key, value string) (string, error) {
	if _, ok := rwiCardinalWidths[key]; ok {
		n := parseRWIDecimalPrefix(value)
		return FormatRWICardinal(fixedWidthUnsigned(n, rwiCardinalWidths[key])), nil
	}
	switch key {
	case ColDocType, ColWordType:
		n := parseRWIDecimalPrefix(value)
		return strconv.FormatUint(uint64(lowByte(n.unsigned())), 10), nil
	case ColLanguage:
		return clampStringBytes(value, langLength), nil
	case ColFlags:
		raw, err := Decode(value)
		if err != nil {
			return "", fmt.Errorf("%w %s: %w", errInvalidRWIProperty, key, err)
		}
		return Encode(clampBytes(raw, rwiByteFlagLength)), nil
	}
	return value, nil
}

func parseRWIDecimalPrefix(value string) rwiDecimalPrefix {
	trimmed := strings.TrimLeft(value, " ")
	if trimmed == "" {
		return rwiDecimalPrefix{}
	}
	pos := 0
	negative := false
	if trimmed[pos] == '-' || trimmed[pos] == '+' {
		negative = trimmed[pos] == '-'
		pos++
		if pos == len(trimmed) {
			return rwiDecimalPrefix{}
		}
	}
	start := pos
	for pos < len(trimmed) && trimmed[pos] >= '0' && trimmed[pos] <= '9' {
		pos++
	}
	if pos == start {
		return rwiDecimalPrefix{}
	}
	magnitude, err := strconv.ParseUint(trimmed[start:pos], 10, 64)
	if err != nil {
		return rwiDecimalPrefix{}
	}
	return rwiDecimalPrefix{magnitude: magnitude, negative: negative}
}

func (n rwiDecimalPrefix) unsigned() uint64 {
	if !n.negative {
		return n.magnitude
	}
	return ^n.magnitude + 1
}

func fixedWidthUnsigned(n rwiDecimalPrefix, width int) uint64 {
	u := n.unsigned()
	if width >= 8 {
		return u
	}
	mask := uint64(1)<<(width*8) - 1
	return u & mask
}

func lowByte(n uint64) byte {
	return byte(n % 256)
}

func clampStringBytes(value string, width int) string {
	if len(value) <= width {
		return value
	}
	return value[:width]
}

func clampBytes(raw []byte, width int) []byte {
	if len(raw) <= width {
		return raw
	}
	return raw[:width]
}

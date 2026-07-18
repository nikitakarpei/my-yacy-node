package yacyproto

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	colURLHash           = "h"
	colLastModified      = "a"
	colFreshUntil        = "s"
	colTitleWordCount    = "u"
	colTextWordCount     = "w"
	colPhraseCount       = "p"
	colDocType           = "d"
	colLanguage          = "l"
	colLocalLinkCount    = "x"
	colExternalLinkCount = "y"
	colURLLength         = "m"
	colURLComponentCount = "n"
	colWordType          = "g"
	colFlags             = "z"
	colHitCount          = "c"
	colTextPosition      = "t"
	colPhraseRelativePos = "r"
	colPhrasePosition    = "o"
	// colWordDistance is a query-time term-distance signal, always 0 in a stored posting; the node derives it from colTextPosition and never reads this column.
	colWordDistance = "i"
	colReserve      = "k"
	propertyOpen    = '{'
	propertyClose   = '}'
)

const (
	rwiByteFlagLength = 4
	langLength        = 2
)

var errInvalidRWIProperty = errors.New("invalid rwi property")

var rwiCardinalWidths = map[string]int{
	colLastModified:      2,
	colFreshUntil:        2,
	colTitleWordCount:    1,
	colTextWordCount:     2,
	colPhraseCount:       2,
	colLocalLinkCount:    1,
	colExternalLinkCount: 1,
	colURLLength:         1,
	colURLComponentCount: 1,
	colHitCount:          1,
	colTextPosition:      2,
	colPhraseRelativePos: 1,
	colPhrasePosition:    1,
	colWordDistance:      1,
	colReserve:           1,
}

var documentTypeByChar = map[byte]yacymodel.DocumentType{
	't': yacymodel.DocumentTypeText,
	'h': yacymodel.DocumentTypeHTML,
	'd': yacymodel.DocumentTypeDocument,
	'i': yacymodel.DocumentTypeImage,
	'm': yacymodel.DocumentTypeMovie,
	'f': yacymodel.DocumentTypeFlash,
	's': yacymodel.DocumentTypeShare,
	'a': yacymodel.DocumentTypeAudio,
	'p': yacymodel.DocumentTypePDF,
	'b': yacymodel.DocumentTypeBinary,
	'u': yacymodel.DocumentTypeUnknown,
}

var charByDocumentType = invertDocumentTypeChars()

func invertDocumentTypeChars() map[yacymodel.DocumentType]byte {
	out := make(map[yacymodel.DocumentType]byte, len(documentTypeByChar))
	for char, documentType := range documentTypeByChar {
		out[documentType] = char
	}
	return out
}

// rwiPostingWireCodec translates between the RWI posting domain type and
// YaCy's DHT property-form line. It is the only thing outside this file that
// needs to know the wire form exists.
type rwiPostingWireCodec struct{}

func (rwiPostingWireCodec) decode(line string) (yacymodel.RWIPosting, error) {
	wire, err := parseRWIPostingLine(line)
	if err != nil {
		return yacymodel.RWIPosting{}, err
	}
	posting, err := wire.domain()
	if err != nil {
		return yacymodel.RWIPosting{}, fmt.Errorf("%w: %w", yacymodel.ErrBadRWIPosting, err)
	}
	return posting, nil
}

func (rwiPostingWireCodec) encode(p yacymodel.RWIPosting) string {
	return rwiPostingWireFormFromDomain(p).line()
}

const maxRWIPostingsPerTransfer = 1000

func (c rwiPostingWireCodec) encodeLines(postings []yacymodel.RWIPosting) string {
	lines := make([]string, len(postings))
	for i, posting := range postings {
		lines[i] = c.encode(posting)
	}

	return strings.Join(lines, "\n")
}

// decodeLines drops postings a peer sent malformed rather than failing the
// whole transfer, and stops reading once the batch limit is reached.
func (c rwiPostingWireCodec) decodeLines(
	ctx context.Context,
	raw string,
) []yacymodel.RWIPosting {
	var postings []yacymodel.RWIPosting
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		if len(postings) >= maxRWIPostingsPerTransfer {
			slog.WarnContext(
				ctx,
				"transfer rwi posting limit reached",
				slog.Int("limit", maxRWIPostingsPerTransfer),
			)
			break
		}

		posting, err := c.decode(line)
		if err != nil {
			slog.WarnContext(
				ctx,
				"transfer rwi posting discarded",
				slog.Any("error", err),
				slog.Int("lineLength", len(line)),
			)
			continue
		}

		postings = append(postings, posting)
	}

	return postings
}

// rwiPostingWireForm is the reverse-word-index posting in its property-form
// wire representation: a flat, protocol-defined column map.
type rwiPostingWireForm struct {
	wordHash   yacymodel.Hash
	properties map[string]string
}

func (e rwiPostingWireForm) urlHash() (yacymodel.URLHash, error) {
	return yacymodel.ParseURLHash(e.properties[colURLHash])
}

func (e rwiPostingWireForm) appearanceFlags() (bitfield, error) {
	value, ok := e.properties[colFlags]
	if !ok {
		return nil, nil
	}
	return decodeBitfield(value)
}

func (e rwiPostingWireForm) byteValue(key string) (byte, error) {
	value := e.properties[key]
	n, err := strconv.ParseUint(value, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("parse rwi byte %s: %w", key, err)
	}
	return byte(n), nil
}

func (e rwiPostingWireForm) uint16Value(key string) (uint16, error) {
	value := e.properties[key]
	n, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("parse rwi uint16 %s: %w", key, err)
	}
	return uint16(n), nil
}

func (e rwiPostingWireForm) optionalByteValue(key string) (byte, error) {
	if _, ok := e.properties[key]; !ok {
		return 0, nil
	}
	return e.byteValue(key)
}

func (e rwiPostingWireForm) optionalUint16Value(key string) (uint16, error) {
	if _, ok := e.properties[key]; !ok {
		return 0, nil
	}
	return e.uint16Value(key)
}

// domain projects the wire form's meaningful columns onto the RWIPosting
// domain concept. Columns that YaCy itself never populates (freshUntil,
// typeofword, worddistance, reserve) are wire-only and have no domain
// counterpart.
func (e rwiPostingWireForm) domain() (yacymodel.RWIPosting, error) {
	urlHash, err := e.urlHash()
	if err != nil {
		return yacymodel.RWIPosting{}, fmt.Errorf("rwi posting url hash: %w", err)
	}

	posting := yacymodel.RWIPosting{
		WordHash: e.wordHash,
		URLHash:  urlHash,
		Language: yacymodel.Language(e.properties[colLanguage]),
	}
	if err := e.fillDomainDate(&posting); err != nil {
		return yacymodel.RWIPosting{}, err
	}
	if err := e.fillDomainCardinals(&posting); err != nil {
		return yacymodel.RWIPosting{}, err
	}
	if err := e.fillDomainDocumentType(&posting); err != nil {
		return yacymodel.RWIPosting{}, err
	}
	if err := e.fillDomainAppearance(&posting); err != nil {
		return yacymodel.RWIPosting{}, err
	}
	return posting, nil
}

func (e rwiPostingWireForm) fillDomainDocumentType(posting *yacymodel.RWIPosting) error {
	char, err := e.optionalByteValue(colDocType)
	if err != nil {
		return fmt.Errorf("rwi posting %s: %w", colDocType, err)
	}
	posting.DocumentType = documentTypeByChar[char]
	return nil
}

func (e rwiPostingWireForm) fillDomainDate(posting *yacymodel.RWIPosting) error {
	value, ok := e.properties[colLastModified]
	if !ok {
		return nil
	}
	lastModified, err := yacymodel.ParseMicroDateWireValue(value)
	if err != nil {
		return fmt.Errorf("rwi posting last modified: %w", err)
	}
	posting.LastModified = lastModified
	return nil
}

func (e rwiPostingWireForm) fillDomainCardinals(posting *yacymodel.RWIPosting) error {
	bytes := []struct {
		column string
		field  *int
	}{
		{colTitleWordCount, &posting.TitleWords},
		{colLocalLinkCount, &posting.LocalLinks},
		{colExternalLinkCount, &posting.ExternalLinks},
		{colURLLength, &posting.URLLength},
		{colURLComponentCount, &posting.URLComponents},
		{colHitCount, &posting.Hits},
		{colPhraseRelativePos, &posting.PhraseRelativePosition},
		{colPhrasePosition, &posting.PhrasePosition},
	}
	for _, b := range bytes {
		value, err := e.optionalByteValue(b.column)
		if err != nil {
			return fmt.Errorf("rwi posting %s: %w", b.column, err)
		}
		*b.field = int(value)
	}
	uint16s := []struct {
		column string
		field  *int
	}{
		{colTextWordCount, &posting.TextWords},
		{colPhraseCount, &posting.Phrases},
		{colTextPosition, &posting.TextPosition},
	}
	for _, u := range uint16s {
		value, err := e.optionalUint16Value(u.column)
		if err != nil {
			return fmt.Errorf("rwi posting %s: %w", u.column, err)
		}
		*u.field = int(value)
	}
	return nil
}

func (e rwiPostingWireForm) fillDomainAppearance(posting *yacymodel.RWIPosting) error {
	flags, err := e.appearanceFlags()
	if err != nil {
		return fmt.Errorf("rwi posting appearance: %w", err)
	}
	if flags != nil {
		posting.Appearance = appearanceFromBitfield(flags)
	}
	return nil
}

// rwiPostingWireFormFromDomain builds the wire form for a domain posting.
// Columns YaCy never populates (freshUntil, typeofword, worddistance,
// reserve) are omitted; a real peer's original values for those columns
// cannot be recovered from the domain type and are not round-tripped.
func rwiPostingWireFormFromDomain(p yacymodel.RWIPosting) rwiPostingWireForm {
	props := map[string]string{
		colURLHash:           p.URLHash.String(),
		colLastModified:      p.LastModified.WireValue(),
		colTitleWordCount:    strconv.Itoa(p.TitleWords),
		colTextWordCount:     strconv.Itoa(p.TextWords),
		colPhraseCount:       strconv.Itoa(p.Phrases),
		colDocType:           strconv.FormatUint(uint64(charByDocumentType[p.DocumentType]), 10),
		colLocalLinkCount:    strconv.Itoa(p.LocalLinks),
		colExternalLinkCount: strconv.Itoa(p.ExternalLinks),
		colURLLength:         strconv.Itoa(p.URLLength),
		colURLComponentCount: strconv.Itoa(p.URLComponents),
		colFlags:             yacymodel.Encode(bitfieldFromAppearance(p.Appearance)),
		colHitCount:          strconv.Itoa(p.Hits),
		colTextPosition:      strconv.Itoa(p.TextPosition),
		colPhraseRelativePos: strconv.Itoa(p.PhraseRelativePosition),
		colPhrasePosition:    strconv.Itoa(p.PhrasePosition),
	}
	if p.Language != "" {
		props[colLanguage] = string(p.Language)
	}
	return rwiPostingWireForm{wordHash: p.WordHash, properties: props}
}

func parseRWIPostingLine(line string) (rwiPostingWireForm, error) {
	open := strings.IndexByte(line, propertyOpen)
	if open < 0 || !strings.HasSuffix(line, string(propertyClose)) {
		return rwiPostingWireForm{}, fmt.Errorf(
			"%w: missing property form",
			yacymodel.ErrBadRWIPosting,
		)
	}
	wordHash, err := yacymodel.ParseHash(line[:open])
	if err != nil {
		return rwiPostingWireForm{}, fmt.Errorf(
			"%w: word hash: %w",
			yacymodel.ErrBadRWIPosting,
			err,
		)
	}
	props, err := yacymodel.ParsePropertyPairs(line[open+1 : len(line)-1])
	if err != nil {
		return rwiPostingWireForm{}, fmt.Errorf("%w: %w", yacymodel.ErrBadRWIPosting, err)
	}
	props, err = normalizeRWIProperties(props)
	if err != nil {
		return rwiPostingWireForm{}, fmt.Errorf("%w: %w", yacymodel.ErrBadRWIPosting, err)
	}
	if err := validateRWIProperties(props); err != nil {
		return rwiPostingWireForm{}, fmt.Errorf("%w: %w", yacymodel.ErrBadRWIPosting, err)
	}
	return rwiPostingWireForm{wordHash: wordHash, properties: props}, nil
}

func (e rwiPostingWireForm) line() string {
	keys := make([]string, 0, len(e.properties))
	for k := range e.properties {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	b.WriteString(string(e.wordHash))
	b.WriteByte(propertyOpen)
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(e.properties[k])
	}
	b.WriteByte(propertyClose)
	return b.String()
}

func formatRWICardinal(value uint64) string {
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
	case colURLHash:
		if _, err := yacymodel.ParseHash(value); err != nil {
			return fmt.Errorf("%w %s: %w", errInvalidRWIProperty, key, err)
		}
	case colLanguage:
		if len(value) != langLength {
			return fmt.Errorf(
				"%w %s: length %d, want %d",
				errInvalidRWIProperty,
				key,
				len(value),
				langLength,
			)
		}
	case colFlags:
		return validateOptionalEncoded(key, value)
	case colDocType, colWordType:
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
	if _, err := yacymodel.Decode(value); err != nil {
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
		return formatRWICardinal(fixedWidthUnsigned(n, rwiCardinalWidths[key])), nil
	}
	switch key {
	case colDocType, colWordType:
		n := parseRWIDecimalPrefix(value)
		return strconv.FormatUint(uint64(lowByte(n.unsigned())), 10), nil
	case colLanguage:
		return clampStringBytes(value, langLength), nil
	case colFlags:
		raw, err := yacymodel.Decode(value)
		if err != nil {
			return "", fmt.Errorf("%w %s: %w", errInvalidRWIProperty, key, err)
		}
		return yacymodel.Encode(clampBytes(raw, rwiByteFlagLength)), nil
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

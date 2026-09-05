package yacyproto

import (
	"context"
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
	colTitleWordCount    = "u"
	colTextWordCount     = "w"
	colPhraseCount       = "p"
	colDocType           = "d"
	colLanguage          = "l"
	colLocalLinkCount    = "x"
	colExternalLinkCount = "y"
	colURLLength         = "m"
	colURLComponentCount = "n"
	colFlags             = "z"
	colHitCount          = "c"
	colTextPosition      = "t"
	colPhraseRelativePos = "r"
	colPhrasePosition    = "o"
	byteColumnMask       = 0xff
	uint16ColumnMask     = 0xffff
	propertyOpen         = '{'
	propertyClose        = '}'
)

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

func (rwiPostingWireCodec) decodePropertyForm(form string) (yacymodel.RWIPosting, error) {
	properties, err := propertyPairsOfRow(form)
	if err != nil {
		return yacymodel.RWIPosting{}, fmt.Errorf("%w: %w", yacymodel.ErrBadRWIPosting, err)
	}
	posting, err := rwiPostingWireForm{properties: properties}.domain()
	if err != nil {
		return yacymodel.RWIPosting{}, fmt.Errorf("%w: %w", yacymodel.ErrBadRWIPosting, err)
	}

	return posting, nil
}

func (rwiPostingWireCodec) encodePropertyForm(p yacymodel.RWIPosting) string {
	return rwiPostingWireFormFromDomain(p).propertyForm()
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

// domain projects the wire form's meaningful columns onto the RWIPosting
// domain concept. A peer also sends freshUntil, typeofword, worddistance and
// reserve columns; this node models none of them and reads past them.
//
// Cardinal columns are read the way YaCy writes them: a decimal prefix, taken
// modulo the column's width. Only a url hash, language or appearance a peer
// mangled beyond repair rejects the posting.
func (e rwiPostingWireForm) domain() (yacymodel.RWIPosting, error) {
	urlHash, err := yacymodel.ParseURLHash(e.properties[colURLHash])
	if err != nil {
		return yacymodel.RWIPosting{}, fmt.Errorf("rwi posting url hash: %w", err)
	}

	language, err := e.language()
	if err != nil {
		return yacymodel.RWIPosting{}, fmt.Errorf("rwi posting language: %w", err)
	}

	appearance, err := e.appearance()
	if err != nil {
		return yacymodel.RWIPosting{}, fmt.Errorf("rwi posting appearance: %w", err)
	}

	return yacymodel.RWIPosting{
		WordHash:               e.wordHash,
		URLHash:                urlHash,
		LastModified:           microDateWireCodec{}.decode(e.cardinal(colLastModified)),
		TitleWords:             int(e.byteCardinal(colTitleWordCount)),
		TextWords:              int(e.uint16Cardinal(colTextWordCount)),
		Phrases:                int(e.uint16Cardinal(colPhraseCount)),
		DocumentType:           documentTypeByChar[e.byteCardinal(colDocType)],
		Language:               language,
		LocalLinks:             int(e.byteCardinal(colLocalLinkCount)),
		ExternalLinks:          int(e.byteCardinal(colExternalLinkCount)),
		URLLength:              int(e.byteCardinal(colURLLength)),
		URLComponents:          int(e.byteCardinal(colURLComponentCount)),
		Appearance:             appearance,
		Hits:                   int(e.byteCardinal(colHitCount)),
		TextPosition:           int(e.uint16Cardinal(colTextPosition)),
		PhraseRelativePosition: int(e.byteCardinal(colPhraseRelativePos)),
		PhrasePosition:         int(e.byteCardinal(colPhrasePosition)),
	}, nil
}

func (e rwiPostingWireForm) cardinal(column string) uint64 {
	value, ok := e.properties[column]
	if !ok {
		return 0
	}
	return parseRWIDecimalPrefix(value).unsigned()
}

func (e rwiPostingWireForm) byteCardinal(column string) byte {
	return byte(e.cardinal(column) & byteColumnMask)
}

func (e rwiPostingWireForm) uint16Cardinal(column string) uint16 {
	return uint16(e.cardinal(column) & uint16ColumnMask)
}

// language keeps only the leading ISO 639-1 code: YaCy peers are known to send
// three-letter codes in this column.
func (e rwiPostingWireForm) language() (yacymodel.Language, error) {
	value := e.properties[colLanguage]
	if value == "" {
		return yacymodel.LanguageOfUndeclaredDocument, nil
	}
	if len(value) > yacymodel.LanguageCodeLength {
		value = value[:yacymodel.LanguageCodeLength]
	}

	return yacymodel.ParseLanguage(value)
}

func (e rwiPostingWireForm) appearance() (yacymodel.Appearance, error) {
	value, ok := e.properties[colFlags]
	if !ok {
		return yacymodel.Appearance{}, nil
	}
	flags, err := decodeBitfield(value)
	if err != nil {
		return yacymodel.Appearance{}, err
	}
	return appearanceFromBitfield(flags), nil
}

// rwiPostingWireFormFromDomain builds the wire form for a domain posting.
// The columns this node does not model (freshUntil, typeofword, worddistance,
// reserve) are omitted: a peer's original values for them cannot be recovered
// from the domain type and are not round-tripped.
func rwiPostingWireFormFromDomain(p yacymodel.RWIPosting) rwiPostingWireForm {
	props := map[string]string{
		colURLHash:           p.URLHash.String(),
		colLastModified:      strconv.FormatUint(microDateWireCodec{}.encode(p.LastModified), 10),
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
	props[colLanguage] = p.Language.String()

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
	props, err := parsePropertyPairs(line[open+1 : len(line)-1])
	if err != nil {
		return rwiPostingWireForm{}, fmt.Errorf("%w: %w", yacymodel.ErrBadRWIPosting, err)
	}
	return rwiPostingWireForm{wordHash: wordHash, properties: props}, nil
}

func (e rwiPostingWireForm) line() string {
	return e.wordHash.String() + e.propertyForm()
}

func (e rwiPostingWireForm) propertyForm() string {
	keys := make([]string, 0, len(e.properties))
	for k := range e.properties {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
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

type rwiDecimalPrefix struct {
	magnitude uint64
	negative  bool
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

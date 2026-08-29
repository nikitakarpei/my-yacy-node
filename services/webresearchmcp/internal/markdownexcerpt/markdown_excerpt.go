// Package markdownexcerpt cuts the markdown of one page down to the number of characters a
// caller asks for, and says how many characters the whole markdown has, so a caller that
// reads only the start of a page knows there is more of it to ask for.
package markdownexcerpt

import "unicode/utf8"

type MarkdownExcerpt struct {
	Markdown               string
	MarkdownCharacterCount int
	Truncated              bool
}

func MarkdownExcerptOf(markdown string, characterLimit int) MarkdownExcerpt {
	markdownCharacterCount := utf8.RuneCountInString(markdown)
	if markdownCharacterCount <= characterLimit {
		return MarkdownExcerpt{
			Markdown:               markdown,
			MarkdownCharacterCount: markdownCharacterCount,
		}
	}
	return MarkdownExcerpt{
		Markdown:               string([]rune(markdown)[:characterLimit]),
		MarkdownCharacterCount: markdownCharacterCount,
		Truncated:              true,
	}
}

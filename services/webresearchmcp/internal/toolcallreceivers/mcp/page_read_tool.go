package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
)

type readPageArguments struct {
	URL            string `json:"url"`
	CharacterLimit int    `json:"characterLimit,omitempty"`
	ToleratedAge   string `json:"toleratedAge,omitempty"   jsonschema:"a duration, such as 30m"`
	Version        string `json:"version,omitempty"        jsonschema:"a version from an earlier answer"`
}

type readPageResult struct {
	URL                    string    `json:"url"`
	Version                string    `json:"version"`
	StoredAt               time.Time `json:"storedAt"`
	FetchOutcome           string    `json:"fetchOutcome"`
	Markdown               string    `json:"markdown"`
	MarkdownCharacterCount int       `json:"markdownCharacterCount"`
	Truncated              bool      `json:"truncated"`
}

type pageReadTool struct {
	pageReader PageReader
	admission  *toolCallAdmission
}

func (t *pageReadTool) readPage(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	arguments readPageArguments,
) (*mcp.CallToolResult, readPageResult, error) {
	if err := t.admission.admit(ctx); err != nil {
		return nil, readPageResult{}, err
	}
	defer t.admission.release()

	pageCall, err := pageCallFrom(arguments)
	if err != nil {
		return nil, readPageResult{}, err
	}
	answer, err := t.pageReader.PageAnswerFor(ctx, pageCall)
	if err != nil {
		return nil, readPageResult{}, err
	}
	return nil, readPageResultFrom(answer), nil
}

func pageCallFrom(arguments readPageArguments) (pageread.PageCall, error) {
	pageURL, err := canonicalurl.CanonicalURLOf(arguments.URL)
	if err != nil {
		return pageread.PageCall{}, fmt.Errorf(
			"read the address %q of the page: %w",
			arguments.URL,
			err,
		)
	}
	toleratedAge, err := toleratedAgeOf(arguments)
	if err != nil {
		return pageread.PageCall{}, err
	}
	return pageread.PageCall{
		URL:            pageURL,
		CharacterLimit: arguments.CharacterLimit,
		ToleratedAge:   toleratedAge,
		Version:        arguments.Version,
	}, nil
}

func toleratedAgeOf(arguments readPageArguments) (time.Duration, error) {
	if arguments.ToleratedAge == "" {
		return 0, nil
	}
	toleratedAge, err := time.ParseDuration(arguments.ToleratedAge)
	if err != nil {
		return 0, fmt.Errorf(
			"read the tolerated age %q of the call: %w",
			arguments.ToleratedAge, err,
		)
	}
	return toleratedAge, nil
}

func readPageResultFrom(answer pageread.PageAnswer) readPageResult {
	return readPageResult{
		URL:                    answer.URL.String(),
		Version:                answer.Version,
		StoredAt:               answer.StoredAt,
		FetchOutcome:           string(answer.FetchOutcome),
		Markdown:               answer.Excerpt.Markdown,
		MarkdownCharacterCount: answer.Excerpt.MarkdownCharacterCount,
		Truncated:              answer.Excerpt.Truncated,
	}
}

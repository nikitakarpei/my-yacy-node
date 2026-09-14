//go:build e2e

package yacypeer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
)

const adminAuthHeader = "Authorization: Basic YWRtaW46eWFjeQ=="

// pushedDocumentTitle holds an ampersand because a peer sends a title on
// without an XML escape, which a reader of the peer must accept.
const pushedDocumentTitle = "yacy rwi node end & to end document"

func TransferTokens() []string {
	tokens := make([]string, 150)
	for i := range tokens {
		tokens[i] = fmt.Sprintf("yacyrwitransferuniquetoken%03d", i)
	}
	return tokens
}

// PushDocument indexes one HTML document of the given tokens into the real
// YaCy peer and reports the address it now holds it under. Only an HTML
// document leaves the peer holding a title for it.
func PushDocument(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL string,
	tokens []string,
) string {
	t.Helper()

	return PushDocumentUnderAddress(
		t, ctx, probe, yacyURL,
		fmt.Sprintf("http://transfer.example.invalid/doc-%d.html", len(tokens)),
		tokens,
	)
}

// PushDocumentUnderAddress indexes one HTML document of the given tokens under
// the address the caller names, so that several peers can hold one document
// with the words each of them indexes for it.
func PushDocumentUnderAddress(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL string,
	documentAddress string,
	tokens []string,
) string {
	t.Helper()

	body, contentType := buildMultipart(
		map[string]string{
			"count":         "1",
			"url-0":         documentAddress,
			"contentType-0": "text/html",
			"collection-0":  "transfer",
			"synchronous":   "true",
			"commit":        "true",
		},
		map[string]string{"data-0": htmlPageOf(tokens)},
	)

	result := probe.PostRaw(ctx, yacyURL+"/api/push_p.json", body,
		"Content-Type: "+contentType, adminAuthHeader)
	if !result.OK {
		t.Fatalf("push_p.json request to YaCy failed: %s", result.Diag())
	}
	if !strings.Contains(result.Body, "successall") {
		t.Fatalf("push_p.json did not report success: %s", result.Body)
	}

	return documentAddress
}

func htmlPageOf(tokens []string) string {
	return "<html><head><title>" + pushedDocumentTitle + "</title></head>" +
		"<body><p>" + strings.Join(tokens, " ") + "</p></body></html>"
}

func buildMultipart(fields, files map[string]string) (body, contentType string) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for key, value := range fields {
		_ = writer.WriteField(key, value)
	}
	for key, value := range files {
		formWriter, _ := writer.CreateFormFile(key, key)
		_, _ = io.WriteString(formWriter, value)
	}
	_ = writer.Close()
	return buf.String(), writer.FormDataContentType()
}

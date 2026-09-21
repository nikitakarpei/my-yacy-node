package judgedqueries_test

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"testing"
	"time"
)

const (
	warcVersionLine             = "WARC/1.1"
	warcRecordTypeField         = "WARC-Type"
	warcTargetAddressField      = "WARC-Target-URI"
	warcRecordDateField         = "WARC-Date"
	warcRecordIdentityField     = "WARC-Record-ID"
	warcBlockTypeField          = "Content-Type"
	warcBlockLengthField        = "Content-Length"
	warcResponseRecordType      = "response"
	warcResponseBlockType       = "application/http;msgtype=response"
	warcRecordEnd               = "\r\n\r\n"
	httpResponseStatusLine      = "HTTP/1.1 200 OK"
	httpContentTypeField        = "Content-Type"
	httpContentLengthField      = "Content-Length"
	httpResponseHeaderEnd       = "\r\n\r\n"
	amountOfRecordIdentityBytes = 16
)

func warcOf(t *testing.T, pages []storedPage) []byte {
	t.Helper()

	var warc bytes.Buffer
	for _, page := range pages {
		warc.Write(warcResponseRecordOf(t, page))
	}

	return warc.Bytes()
}

func warcResponseRecordOf(t *testing.T, page storedPage) []byte {
	t.Helper()

	block := httpResponseBlockOf(page)
	var record bytes.Buffer
	record.WriteString(warcVersionLine + "\r\n")
	for _, field := range [][2]string{
		{warcRecordTypeField, warcResponseRecordType},
		{warcTargetAddressField, page.address},
		{warcRecordDateField, time.Now().UTC().Format(time.RFC3339)},
		{warcRecordIdentityField, warcRecordIdentity(t)},
		{warcBlockTypeField, warcResponseBlockType},
		{warcBlockLengthField, strconv.Itoa(len(block))},
	} {
		fmt.Fprintf(&record, "%s: %s\r\n", field[0], field[1])
	}
	record.WriteString("\r\n")
	record.Write(block)
	record.WriteString(warcRecordEnd)

	return record.Bytes()
}

func httpResponseBlockOf(page storedPage) []byte {
	var block bytes.Buffer
	block.WriteString(httpResponseStatusLine + "\r\n")
	fmt.Fprintf(&block, "%s: %s\r\n", httpContentTypeField, page.contentType)
	fmt.Fprintf(&block, "%s: %d", httpContentLengthField, len(page.body))
	block.WriteString(httpResponseHeaderEnd)
	block.Write(page.body)

	return block.Bytes()
}

func warcRecordIdentity(t *testing.T) string {
	t.Helper()

	identity := make([]byte, amountOfRecordIdentityBytes)
	if _, err := rand.Read(identity); err != nil {
		t.Fatalf("warc record identity: %v", err)
	}

	return fmt.Sprintf(
		"<urn:uuid:%s-%s-%s-%s-%s>",
		hex.EncodeToString(identity[0:4]),
		hex.EncodeToString(identity[4:6]),
		hex.EncodeToString(identity[6:8]),
		hex.EncodeToString(identity[8:10]),
		hex.EncodeToString(identity[10:16]),
	)
}

type warcRecord struct {
	fields textproto.MIMEHeader
	block  []byte
}

func pagesOfWARC(t *testing.T, warc []byte) []storedPage {
	t.Helper()

	var pages []storedPage
	reader := bufio.NewReader(bytes.NewReader(warc))
	for {
		record, read := nextWARCRecordIn(t, reader)
		if !read {
			return pages
		}
		if record.fields.Get(warcRecordTypeField) != warcResponseRecordType {
			continue
		}
		pages = append(pages, pageOf(t, record))
	}
}

func nextWARCRecordIn(t *testing.T, reader *bufio.Reader) (warcRecord, bool) {
	t.Helper()

	if !readWARCVersionLine(t, reader) {
		return warcRecord{}, false
	}
	fields, err := textproto.NewReader(reader).ReadMIMEHeader()
	if err != nil {
		t.Fatalf("read a warc record: %v", err)
	}
	blockLength, err := strconv.Atoi(fields.Get(warcBlockLengthField))
	if err != nil {
		t.Fatalf("read a warc record: %s: %v", warcBlockLengthField, err)
	}
	block := make([]byte, blockLength)
	if _, err := io.ReadFull(reader, block); err != nil {
		t.Fatalf("read a warc record: %v", err)
	}
	if _, err := io.CopyN(io.Discard, reader, int64(len(warcRecordEnd))); err != nil {
		t.Fatalf("read a warc record: %v", err)
	}

	return warcRecord{fields: fields, block: block}, true
}

func readWARCVersionLine(t *testing.T, reader *bufio.Reader) bool {
	t.Helper()

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF && len(bytes.TrimSpace([]byte(line))) == 0 {
			return false
		}
		if err != nil && err != io.EOF {
			t.Fatalf("read a warc record: %v", err)
		}
		trimmed := string(bytes.TrimSpace([]byte(line)))
		if trimmed == "" {
			continue
		}
		if trimmed != warcVersionLine {
			t.Fatalf("read a warc record: the version line reads %q, want %q",
				trimmed, warcVersionLine)
		}

		return true
	}
}

func pageOf(t *testing.T, record warcRecord) storedPage {
	t.Helper()

	response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(record.block)), nil)
	if err != nil {
		t.Fatalf("read a warc response record: %v", err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read a warc response record: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("read a warc response record: %v", err)
	}

	return storedPage{
		address:     record.fields.Get(warcTargetAddressField),
		contentType: response.Header.Get(httpContentTypeField),
		body:        body,
	}
}

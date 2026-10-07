# Judged query pages

The pages of the judged queries are in the private repository
`judged-query-pages`. Clone it to `test/judgedqueries/testdata/pages/`. The gate
does not read it. Commit a change to the pages in that repository.

Each query has two files, named by the query words in lower case and joined by
`-`:

- `<query>.warc.zst` holds the pages as a capture read them. The derivation
  reads them.
- `<query>.evidence.warc.zst` holds copies of pages that a site refuses to the
  capture, taken by a browser or a web archive. Only graders read them.

Each record keeps the time of its capture, the HTTP status and, for a copy, the
tool that took it.

## How to capture the pages again

The capture reads from the web the page of each document a judgments file
names, and writes the pages file of every query again. It needs egress.

```sh
YACYDHTSEARCH_CAPTURE_JUDGED_QUERY_PAGES=1 go test -timeout 40m -v \
    -run TestCaptureThePagesOfTheJudgedQueries ./test/judgedqueries/
```

## How to capture the missing pages

This step reads only the judged documents that have no stored page. It adds
the pages it gets to the end of the pages file and keeps the earlier pages. It
derives their contents and keeps the reason of each failure in the findings
file: `passing`, `refused` or `gone`. It applies the rules of
`judged-query-grades.md` for `passing` and `gone`. It needs egress.

```sh
YACYDHTSEARCH_CAPTURE_MISSING_JUDGED_QUERY_PAGES=1 go test -timeout 4h -v \
    -run TestCaptureTheMissingPagesOfTheJudgedQueries ./test/judgedqueries/
```

## How to add copies of refused pages

This step reads a directory of JSON files. Each file holds `key`
(`<query>__<document hash>`), `address`, `contentType`, `body` in base64,
`capturedAt`, `capturedBy` and `status`. It adds each new copy to the evidence
file of its query. A copy with the status `404` or `410` marks the document
`gone`. A new copy of a page that is not gone removes the grade of its document.

```sh
YACYDHTSEARCH_IMPORT_JUDGED_QUERY_EVIDENCE=<directory> go test -v \
    -run TestImportTheEvidencePagesOfTheJudgedQueries ./test/judgedqueries/
```

## How to list the documents to grade

This step writes one JSON line for each judged document without a grade,
except a document whose capture failed for a passing reason or whose page is
gone. Each line holds the query, its language, the address, the title, the
snippet of the peers, and the text of the stored page or of the copy.

```sh
YACYDHTSEARCH_DUMP_UNGRADED_JUDGED_DOCUMENTS=<file> go test -v \
    -run TestDumpTheUngradedJudgedDocuments ./test/judgedqueries/
```

package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	webarchivespywb "github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/internal/webarchives/pywb"
)

const (
	EnvScrapeRequestNATSURL = "SCRAPE_REQUEST_NATS_URL"

	FlagPywbURL        = "pywb-url"
	FlagPywbCollection = "pywb-collection"
	FlagURL            = "url"
	FlagMatchType      = "match-type"
	FlagFrom           = "from"
	FlagTo             = "to"
	FlagPageLimit      = "page-limit"
	FlagDryRun         = "dry-run"

	DefaultMatchType = "domain"

	ScrapedMediaType  = "text/html"
	ScrapedStatusCode = http.StatusOK
)

type CommandConfig struct {
	PywbURL              *url.URL
	PywbCollection       string
	PywbCaptureQueries   []webarchivespywb.CaptureQuery
	PageLimit            int
	ScrapeRequestNATSURL string
	DryRun               bool
}

func LoadCommandConfig(
	arguments []string,
	getenv func(string) string,
) (CommandConfig, error) {
	flags := flag.NewFlagSet("webarchivescrape", flag.ContinueOnError)
	pywbURL := flags.String(FlagPywbURL, "", "base address of the archive, as http://pywb:8080")
	pywbCollection := flags.String(
		FlagPywbCollection,
		"",
		"collection the archive publishes, as imported",
	)
	queried := queriedURLs{}
	flags.Var(&queried, FlagURL, "site or page the archive is asked about; state it once for each")
	matchType := flags.String(
		FlagMatchType,
		DefaultMatchType,
		"how far a -url reaches: exact one page, prefix one part of a site,\n"+
			"host one host, domain a site and its subdomains",
	)
	from := flags.String(FlagFrom, "", "earliest capture to select, as 20240101 or 20240101120000")
	to := flags.String(FlagTo, "", "latest capture to select, as 20240101 or 20240101120000")
	pageLimit := flags.Int(
		FlagPageLimit,
		0,
		"most pages one run publishes over all urls together; zero means all",
	)
	dryRun := flags.Bool(
		FlagDryRun,
		false,
		"write the pages that would be published, and publish nothing",
	)
	if err := flags.Parse(arguments); err != nil {
		return CommandConfig{}, fmt.Errorf("read arguments: %w", err)
	}

	parsedPywbURL, err := parsedFlagURL(FlagPywbURL, *pywbURL)
	if err != nil {
		return CommandConfig{}, err
	}
	if strings.TrimSpace(*pywbCollection) == "" {
		return CommandConfig{}, fmt.Errorf("-%s is required", FlagPywbCollection)
	}
	if len(queried) == 0 {
		return CommandConfig{}, fmt.Errorf("-%s is required", FlagURL)
	}
	if *pageLimit < 0 {
		return CommandConfig{}, fmt.Errorf("-%s must not be negative", FlagPageLimit)
	}
	natsURL := getenv(EnvScrapeRequestNATSURL)
	if !*dryRun && strings.TrimSpace(natsURL) == "" {
		return CommandConfig{}, fmt.Errorf(
			"%s is required unless -%s is given", EnvScrapeRequestNATSURL, FlagDryRun,
		)
	}

	return CommandConfig{
		PywbURL:              parsedPywbURL,
		PywbCollection:       *pywbCollection,
		PywbCaptureQueries:   captureQueriesFor(queried, *matchType, *from, *to),
		PageLimit:            *pageLimit,
		ScrapeRequestNATSURL: natsURL,
		DryRun:               *dryRun,
	}, nil
}

type queriedURLs []string

func (u *queriedURLs) String() string {
	return strings.Join(*u, " ")
}

func (u *queriedURLs) Set(queried string) error {
	if strings.TrimSpace(queried) == "" {
		return fmt.Errorf("-%s must name a url", FlagURL)
	}
	*u = append(*u, queried)
	return nil
}

func captureQueriesFor(
	queried queriedURLs,
	matchType string,
	from string,
	to string,
) []webarchivespywb.CaptureQuery {
	queries := make([]webarchivespywb.CaptureQuery, 0, len(queried))
	for _, queriedURL := range queried {
		queries = append(queries, webarchivespywb.CaptureQuery{
			URL:        queriedURL,
			MatchType:  matchType,
			MediaType:  ScrapedMediaType,
			StatusCode: ScrapedStatusCode,
			From:       from,
			To:         to,
		})
	}
	return queries
}

func parsedFlagURL(name string, raw string) (*url.URL, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("-%s is required", name)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("read -%s %q: %w", name, raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("-%s %q must be an http or https address", name, raw)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("-%s %q names no host", name, raw)
	}
	return parsed, nil
}

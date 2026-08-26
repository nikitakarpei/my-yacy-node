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
	PywbCaptureQuery     webarchivespywb.CaptureQuery
	PageLimit            int
	ScrapeRequestNATSURL string
	DryRun               bool
}

func LoadCommandConfig(
	arguments []string,
	getenv func(string) string,
) (CommandConfig, error) {
	flags := flag.NewFlagSet("webarchivescrape", flag.ContinueOnError)
	pywbURL := flags.String(FlagPywbURL, "", "base address of the pywb instance")
	pywbCollection := flags.String(FlagPywbCollection, "", "collection inside pywb")
	queried := flags.String(FlagURL, "", "url the archive is asked about")
	matchType := flags.String(FlagMatchType, DefaultMatchType, "exact, prefix, host, or domain")
	from := flags.String(FlagFrom, "", "earliest capture timestamp")
	to := flags.String(FlagTo, "", "latest capture timestamp")
	pageLimit := flags.Int(
		FlagPageLimit,
		0,
		"most distinct archived pages to publish; zero means all",
	)
	dryRun := flags.Bool(FlagDryRun, false, "write the scrape requests instead of publishing")
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
	if strings.TrimSpace(*queried) == "" {
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
		PywbURL:        parsedPywbURL,
		PywbCollection: *pywbCollection,
		PywbCaptureQuery: webarchivespywb.CaptureQuery{
			URL:        *queried,
			MatchType:  *matchType,
			MediaType:  ScrapedMediaType,
			StatusCode: ScrapedStatusCode,
			From:       *from,
			To:         *to,
		},
		PageLimit:            *pageLimit,
		ScrapeRequestNATSURL: natsURL,
		DryRun:               *dryRun,
	}, nil
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

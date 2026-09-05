//go:build e2e

// Package peerdirectory parses YaCy's peer-directory XML views (seedlist,
// network) into hash sets.
package peerdirectory

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

func SeniorHashes(body []byte) (map[string]struct{}, error) {
	var doc struct {
		Seeds []struct {
			Hash     string `xml:"Hash"`
			PeerType string `xml:"PeerType"`
		} `xml:"seed"`
	}
	out := map[string]struct{}{}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return out, fmt.Errorf("parse seed list: %w", err)
	}
	for _, seed := range doc.Seeds {
		if seed.Hash != "" && seed.PeerType == "senior" {
			out[seed.Hash] = struct{}{}
		}
	}
	return out, nil
}

func ActivePeerHashes(body []byte) (map[string]struct{}, error) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	out := map[string]struct{}{}
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return map[string]struct{}{}, fmt.Errorf("read peer directory: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "hash" {
			continue
		}
		var hash string
		if err := decoder.DecodeElement(&hash, &start); err != nil {
			return map[string]struct{}{}, fmt.Errorf("read peer hash: %w", err)
		}
		hash = strings.TrimSpace(hash)
		if hash != "" {
			out[hash] = struct{}{}
		}
	}
	return out, nil
}

// IndexedWordCounts reports the ICount each seed publishes, the RWI word count
// YaCy reads to decide whether a peer is worth a remote search
// (DHTSelection.java:215).
func IndexedWordCounts(body []byte) (map[string]int, error) {
	var doc struct {
		Seeds []struct {
			Hash   string `xml:"Hash"`
			ICount int    `xml:"ICount"`
		} `xml:"seed"`
	}
	out := map[string]int{}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return out, fmt.Errorf("parse seed list: %w", err)
	}
	for _, seed := range doc.Seeds {
		if seed.Hash != "" {
			out[seed.Hash] = seed.ICount
		}
	}
	return out, nil
}

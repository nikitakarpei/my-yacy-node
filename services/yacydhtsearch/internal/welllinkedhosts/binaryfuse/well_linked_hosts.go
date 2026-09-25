// Package binaryfuse holds the well-linked hosts in a binary fuse filter that
// it builds from a list of host names, one per line. It compares hosts by their
// site, so a listed host also holds its address with or without the world wide
// web label. A small share of the hosts outside the list is held by mistake.
package binaryfuse

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"io"
	"strings"

	"github.com/FastFilter/xorfilter"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type WellLinkedHosts struct {
	filter        *xorfilter.BinaryFuse[uint8]
	amountOfHosts int
}

func New(hostList io.Reader) (WellLinkedHosts, error) {
	siteKeys, err := siteKeysIn(hostList)
	if err != nil {
		return WellLinkedHosts{}, err
	}
	amountOfHosts := len(siteKeys)
	filter, err := xorfilter.NewBinaryFuse[uint8](siteKeys)
	if err != nil {
		return WellLinkedHosts{}, fmt.Errorf("hold the listed hosts: %w", err)
	}

	return WellLinkedHosts{filter: filter, amountOfHosts: amountOfHosts}, nil
}

func siteKeysIn(hostList io.Reader) ([]uint64, error) {
	var siteKeys []uint64
	lines := bufio.NewScanner(hostList)
	for lines.Scan() {
		host := strings.TrimSpace(lines.Text())
		if host == "" {
			continue
		}
		siteKeys = append(siteKeys, siteKeyOf(yacymodel.SiteOfHost(host)))
	}

	if err := lines.Err(); err != nil {
		return nil, fmt.Errorf("read the host list: %w", err)
	}

	return siteKeys, nil
}

func siteKeyOf(site string) uint64 {
	siteHash := fnv.New64a()
	_, _ = siteHash.Write([]byte(site))

	return siteHash.Sum64()
}

func (hosts WellLinkedHosts) HoldsHostOf(address string) bool {
	return hosts.filter.Contains(siteKeyOf(yacymodel.SiteOf(address)))
}

func (hosts WellLinkedHosts) AmountOfHosts() int {
	return hosts.amountOfHosts
}

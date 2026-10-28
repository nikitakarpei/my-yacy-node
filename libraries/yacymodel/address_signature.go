package yacymodel

import (
	"crypto/ed25519"
	"encoding/binary"
	"slices"
	"strconv"
	"time"
)

const addressSignatureDomain = "yacy-address-signature/1"

type AddressSignature struct {
	IssuedAt  time.Time
	Signature [ed25519.SignatureSize]byte
}

func ParseSignature(text string) ([ed25519.SignatureSize]byte, error) {
	var signature [ed25519.SignatureSize]byte
	err := decodeExactly(text, signature[:])

	return signature, err
}

func addressClaimOf(seed Seed, networkName string, issuedAt time.Time) []byte {
	claim := []byte(addressSignatureDomain)
	claim = appendClaimField(claim, networkName)
	claim = appendClaimField(claim, seed.Hash.String())
	hosts := signedHostsOf(seed)
	claim = appendClaimField(claim, strconv.Itoa(len(hosts)))
	for _, host := range hosts {
		claim = appendClaimField(claim, host)
	}
	claim = appendClaimField(claim, portTextOf(seed.Port))
	claim = appendClaimField(claim, portTextOf(seed.announcedSecurePort()))

	return appendClaimField(claim, strconv.FormatInt(issuedAt.Unix(), 10))
}

func appendClaimField(claim []byte, field string) []byte {
	claim = binary.AppendUvarint(claim, uint64(len(field)))

	return append(claim, field...)
}

func signedHostsOf(seed Seed) []string {
	seedHosts := seed.hosts()
	hosts := make([]string, 0, len(seedHosts))
	for _, host := range seedHosts {
		hosts = append(hosts, host.normalized())
	}
	slices.Sort(hosts)

	return slices.Compact(hosts)
}

func portTextOf(port Optional[Port]) string {
	if present, ok := port.Get(); ok {
		return present.String()
	}

	return ""
}

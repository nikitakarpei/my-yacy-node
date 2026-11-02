// Package asmap maps an IP address to the number of the autonomous system that
// announces it, from the AS map that Bitcoin Core ships. A private or other
// unroutable address maps to ASN 0, and so does public space the map leaves
// unassigned.
package asmap

import (
	_ "embed"
	"net/netip"
)

//go:embed ip_asn.dat
var bitcoinCoreProgram []byte

type ASN uint32

const unassignedASN ASN = 0

type Map struct {
	program []byte
}

func BitcoinCore() Map {
	return Map{program: bitcoinCoreProgram}
}

func (m Map) ASNOf(address netip.Addr) ASN {
	if !isRoutable(address) {
		return unassignedASN
	}

	return m.mappedASNOf(address)
}

func isRoutable(address netip.Addr) bool {
	return address.IsGlobalUnicast() && !address.IsPrivate()
}

func (m Map) mappedASNOf(address netip.Addr) ASN {
	program := &programCursor{program: m.program}
	bits := &addressCursor{address: address.As16()}

	defaultASN := unassignedASN
	for {
		switch program.readInstruction() {
		case returnInstruction:
			return program.readASN()
		case jumpInstruction:
			jump := program.readJump()
			if bits.readBit() {
				program.skip(jump)
			}
		case matchInstruction:
			if !bits.readMatching(program.readMatch()) {
				return defaultASN
			}
		case defaultInstruction:
			defaultASN = program.readASN()
		}
	}
}

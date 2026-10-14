package messageadmission

import (
	"net/netip"
	"sync"
	"time"
)

const (
	messagesPerHour          = 20
	messagesPerSenderPerHour = 5
	ipv6SenderPrefixBits     = 64
)

type messageAllowance struct {
	mu         sync.Mutex
	now        func() time.Time
	admissions []admission
}

type admission struct {
	sender string
	at     time.Time
}

func newMessageAllowance(now func() time.Time) *messageAllowance {
	return &messageAllowance{now: now}
}

func (a *messageAllowance) take(address string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := a.now()
	a.forgetAdmissionsBefore(now.Add(-time.Hour))

	sender := senderOf(address)
	if len(a.admissions) >= messagesPerHour || a.amountFrom(sender) >= messagesPerSenderPerHour {
		return false
	}

	a.admissions = append(a.admissions, admission{sender: sender, at: now})

	return true
}

func (a *messageAllowance) forgetAdmissionsBefore(cutoff time.Time) {
	recent := a.admissions[:0]
	for _, admitted := range a.admissions {
		if admitted.at.After(cutoff) {
			recent = append(recent, admitted)
		}
	}
	a.admissions = recent
}

func senderOf(address string) string {
	parsed, err := netip.ParseAddr(address)
	if err != nil {
		return address
	}

	parsed = parsed.Unmap()
	if parsed.Is4() {
		return parsed.String()
	}

	return netip.PrefixFrom(parsed, ipv6SenderPrefixBits).Masked().String()
}

func (a *messageAllowance) amountFrom(sender string) int {
	amount := 0
	for _, admitted := range a.admissions {
		if admitted.sender == sender {
			amount++
		}
	}

	return amount
}

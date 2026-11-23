package utils

import (
	"fmt"
	"net/netip"
	"strings"
)

// NormalizeCIDR adds a host prefix to a plain IP address: /32 for IPv4 and
// /128 for IPv6. Values that already have a prefix, or are not IP addresses,
// are returned trimmed but otherwise unchanged.
func NormalizeCIDR(address string) string {
	address = strings.TrimSpace(address)
	if strings.Contains(address, "/") {
		return address
	}

	ip, err := netip.ParseAddr(address)
	if err != nil {
		return address
	}

	return fmt.Sprintf("%s/%d", address, ip.BitLen())
}

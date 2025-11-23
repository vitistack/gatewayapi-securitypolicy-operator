package utils

import (
	"net"
	"strings"
)

func CheckValidCIDR(ip string) bool {
	_, _, parsedCIDR := net.ParseCIDR(strings.TrimSpace(ip))
	if parsedCIDR == nil {
		return true
	} else {
		return false
	}
}

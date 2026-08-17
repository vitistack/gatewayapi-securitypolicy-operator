package utils

import (
	"net"
	"strings"
)

func CheckValidCIDR(ip string) bool {
	_, _, err := net.ParseCIDR(strings.TrimSpace(ip))
	return err == nil
}

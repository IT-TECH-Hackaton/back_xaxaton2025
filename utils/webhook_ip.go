package utils

import (
	"net"
	"strings"
)

// YooKassaWebhookIPs — официальный whitelist IP ЮKassa для webhook
// https://yookassa.ru/developers/using-api/webhooks
var yooKassaIPRanges = []string{
	"77.75.156.35",
	"77.75.156.11",
	"77.75.154.128/25",
	"77.75.153.0/25",
	"185.71.77.0/27",
	"185.71.76.0/27",
	"2a02:5180::/32",
}

// IsYooKassaIP проверяет, что clientIP принадлежит whitelist ЮKassa
func IsYooKassaIP(clientIP string) bool {
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil {
		return false
	}
	for _, cidr := range yooKassaIPRanges {
		if strings.Contains(cidr, "/") {
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				continue
			}
			if network.Contains(ip) {
				return true
			}
		} else {
			allowed := net.ParseIP(cidr)
			if allowed != nil && ip.Equal(allowed) {
				return true
			}
		}
	}
	return false
}

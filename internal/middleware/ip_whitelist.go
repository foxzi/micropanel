package middleware

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

// IPWhitelist returns middleware that restricts access to allowed IPs/CIDRs.
// If allowedIPs is empty, all IPs are BLOCKED (secure default).
// Use IPWhitelistOptional for cases where empty whitelist should allow all.
func IPWhitelist(allowedIPs []string) gin.HandlerFunc {
	return ipWhitelistMiddleware(allowedIPs, false)
}

// IPWhitelistOptional returns middleware that restricts access to allowed IPs/CIDRs.
// If allowedIPs is empty, all IPs are ALLOWED (for optional restrictions).
func IPWhitelistOptional(allowedIPs []string) gin.HandlerFunc {
	return ipWhitelistMiddleware(allowedIPs, true)
}

func ipWhitelistMiddleware(allowedIPs []string, allowEmptyList bool) gin.HandlerFunc {
	// Parse CIDRs once at initialization
	var networks []*net.IPNet
	var ips []net.IP

	for _, entry := range allowedIPs {
		// Try parsing as CIDR first
		_, network, err := net.ParseCIDR(entry)
		if err == nil {
			networks = append(networks, network)
			continue
		}

		// Try parsing as single IP
		ip := net.ParseIP(entry)
		if ip != nil {
			ips = append(ips, ip)
		}
	}

	return func(c *gin.Context) {
		// Handle empty whitelist based on mode
		if len(networks) == 0 && len(ips) == 0 {
			if allowEmptyList {
				c.Next()
				return
			}
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		clientIP := net.ParseIP(c.ClientIP())
		if clientIP == nil {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Check against single IPs
		for _, ip := range ips {
			if ip.Equal(clientIP) {
				c.Next()
				return
			}
		}

		// Check against CIDR networks
		for _, network := range networks {
			if network.Contains(clientIP) {
				c.Next()
				return
			}
		}

		c.AbortWithStatus(http.StatusForbidden)
	}
}

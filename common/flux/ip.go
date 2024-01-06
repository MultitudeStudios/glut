package flux

import (
	"net"
	"net/http"
	"strings"
)

// IPExtractor is a function to extract IP address from http.Request.
// Set appropriate one to Server.IPExtractor.
type IPExtractor func(*http.Request) string

// ExtractIPDirect extracts IP address using actual IP address.
// Use this if server is directly exposed to the internet (i.e.: uses no proxy).
func ExtractIPDirect() IPExtractor {
	return extractIP
}

// extractIP extracts IP address directly from the http.Request.
func extractIP(req *http.Request) string {
	ra, _, _ := net.SplitHostPort(req.RemoteAddr)
	return ra
}

// ExtractIPFromRealIPHeader extracts IP address using x-real-ip header.
// Use this if you use a proxy that uses this header.
func ExtractIPFromRealIPHeader() IPExtractor {
	return func(req *http.Request) string {
		realIP := req.Header.Get(HeaderXRealIP)
		if realIP != "" {
			realIP = strings.TrimPrefix(realIP, "[")
			realIP = strings.TrimSuffix(realIP, "]")
			if ip := net.ParseIP(realIP); ip != nil {
				return realIP
			}
		}
		return extractIP(req)
	}
}

// ExtractIPFromXFFHeader extracts IP address using x-forwarded-for header.
// Use this if you use a proxy that uses this header.
// If all IPs are trustable, returns furthest one (i.e.: XFF[0]).
func ExtractIPFromXFFHeader() IPExtractor {
	return func(req *http.Request) string {
		directIP := extractIP(req)
		xffs := req.Header[HeaderXForwardedFor]
		if len(xffs) == 0 {
			return directIP
		}
		ips := append(strings.Split(strings.Join(xffs, ","), ","), directIP)
		for i := len(ips) - 1; i >= 0; i-- {
			ips[i] = strings.TrimSpace(ips[i])
			ips[i] = strings.TrimPrefix(ips[i], "[")
			ips[i] = strings.TrimSuffix(ips[i], "]")
			ip := net.ParseIP(ips[i])
			if ip == nil {
				// Unable to parse IP; return direct IP.
				return directIP
			}
		}
		// All of the IPs are trusted; return first element because it is furthest from server (best effort strategy).
		return strings.TrimSpace(ips[0])
	}
}

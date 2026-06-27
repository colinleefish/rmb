package config

import "net"

// addrRequiresAuth reports whether the listen address is reachable off localhost.
// Empty host or 0.0.0.0 binds all interfaces; any non-loopback host needs auth.
func addrRequiresAuth(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return true
	}
	host = stripBrackets(host)
	switch host {
	case "", "0.0.0.0", "::":
		return true
	case "127.0.0.1", "localhost", "::1":
		return false
	default:
		return true
	}
}

func stripBrackets(host string) string {
	if len(host) >= 2 && host[0] == '[' && host[len(host)-1] == ']' {
		return host[1 : len(host)-1]
	}
	return host
}

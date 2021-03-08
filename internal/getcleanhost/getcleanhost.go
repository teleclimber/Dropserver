package getcleanhost

import (
	"net"
	"strings"
)

// GetCleanHost returns the host in a host-port string.
// If there is no port it returns the passed string.
func GetCleanHost(hostPort string) (string, error) {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		addrErr, valid := err.(*net.AddrError)
		if valid && addrErr.Err == "missing port in address" {
			host = hostPort
		} else {
			return "", err
		}
	}
	host = strings.ToLower(host)

	return host, nil
}

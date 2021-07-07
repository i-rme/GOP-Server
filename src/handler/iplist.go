package handler

import (
	"net"
	"net/http"
	"pfg/src/server/config"
	"pfg/src/server/logs"
)

// Denied tries to handle denied requests
func Denied(w http.ResponseWriter, r *http.Request) {

	r.URL.Path = config.IPDeniedScript

	Handle(w, r)

	logs.WriteError("IP address is blocked for accessing this site.")

}

// IsDenied tries to match the remote IP from the request to our deny list
func IsDenied(r *http.Request) bool {

	if r.URL.Path == config.IPDeniedScript {
		return false
	}

	return IsDeniedIP(r.RemoteAddr)

}

// IsDeniedIP tries to match the remote IP to our deny list
func IsDeniedIP(remoteAddr string) bool {

	ip, _, _ := net.SplitHostPort(remoteAddr) // Removes the port from the IP

	for _, ipRange := range config.DenyFrom {
		if IsInSubnet(ip, ipRange) {
			return true
		}
	}

	return false

}

// IsInSubnet tries to match the IP to an ip range or subnet
func IsInSubnet(ipAddr string, ipRange string) bool {

	ip := net.ParseIP(ipAddr) // IP string to net.IP type
	_, ipNetwork, _ := net.ParseCIDR(ipRange)

	return ipNetwork.Contains(ip)
}

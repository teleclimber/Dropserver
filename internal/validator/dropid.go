package validator

import (
	"strings"
)

// JoinDropID joins the domain and handle of a dropid
func JoinDropID(handle, domain string) string {
	return domain + "/" + handle
}

// SplitDropID splits the dropid key into its domain and handle subparts
func SplitDropID(key string) (handle, domain string) {
	pieces := strings.SplitN(key, "/", 2)
	if len(pieces) > 0 {
		domain = pieces[0]
	}
	if len(pieces) > 1 {
		handle = pieces[1]
	}
	return
}

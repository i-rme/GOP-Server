package handler

import (
	"net/http"
	"pfg/src/server/config"
	"strings"
)

// Alias tries to handle aliased requests
func Alias(w http.ResponseWriter, r *http.Request) {
	urlPath := GetURLPath(r) // Obtains the url from the request
	_, aliasedPath := GetAlias(urlPath)

	r.URL.Path = aliasedPath

	Handle(w, r)

}

// IsAlias tries to match urlPath with the aliases defined
func IsAlias(urlPath string) bool {

	for alias := range config.Aliases {
		if strings.HasPrefix("/"+urlPath, alias) {
			return true
		}
	}

	return false

}

// GetAlias tries to match urlPath with the aliases defined
func GetAlias(urlPath string) (string, string) {

	for key, value := range config.Aliases {
		if strings.HasPrefix("/"+urlPath, key) {
			return key, value
		}
	}

	return "", ""

}

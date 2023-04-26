package utils

import (
	"errors"

	log "github.com/Sirupsen/logrus"

	// "log"
	"net/url"
)

// ValidateHosts takes a list of potential RGW urls and returns a list
// of hosts at least adhere to the expected URL format
// Hosts must be http or https
func ValidateHosts(candidateHosts []string) ([]string, error) {
	var validHosts []string
	var err error = nil

	for _, hostURL := range candidateHosts {
		u, err := url.Parse(hostURL)
		if (err != nil) || (u.Scheme != "http" && u.Scheme != "https") {
			log.Warningf("- dropping invalid URL '%s'", hostURL)
			continue
		}
		validHosts = append(validHosts, hostURL)
	}

	if len(validHosts) == 0 {
		err = errors.New("no valid endpoints provided")
	}
	return validHosts, err
}

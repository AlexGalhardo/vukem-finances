package workspaces

import (
	"strings"
	"time"
	// The IANA database travels in the binary: a slim container has no zoneinfo files.
	_ "time/tzdata"
)

// DefaultTimezone is the timezone of a new workspace.
const DefaultTimezone = "UTC"

// LoadZone returns the location of an IANA name; "" is UTC. An unknown name
// (and "Local", which would follow the server) is an error.
func LoadZone(name string) (*time.Location, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return time.UTC, nil
	}

	if strings.EqualFold(name, "local") {
		return nil, errBadTimezone
	}

	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, errBadTimezone
	}

	return loc, nil
}

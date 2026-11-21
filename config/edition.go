package config

import (
	"fmt"
	"strings"

	vconfig "github.com/AlexGalhardo/vukem/config"
)

// Edition is the shape the binary runs in (APP_EDITION, decision D-003).
type Edition string

// The two editions of the application.
const (
	// EditionPersonal is one implicit workspace, no billing, no public
	// sign-up and no landing page.
	EditionPersonal Edition = "personal"
	// EditionSaaS is the multi-tenant service: public sign-up, many
	// workspaces, billing and plan limits.
	EditionSaaS Edition = "saas"
)

// LoadEdition reads APP_EDITION from src. Unset means personal, the safe
// default for a self-hoster: nothing is exposed or billed. Any other value
// is an error, so a typo never boots the wrong edition.
func LoadEdition(src vconfig.Lookuper) (Edition, error) {
	v, _ := src.Lookup("APP_EDITION")

	switch e := Edition(strings.TrimSpace(v)); e {
	case "", EditionPersonal:
		return EditionPersonal, nil
	case EditionSaaS:
		return EditionSaaS, nil
	default:
		return "", fmt.Errorf("APP_EDITION=%q is invalid: use %q or %q", v, EditionPersonal, EditionSaaS)
	}
}

// Personal reports whether e is the personal edition.
func (e Edition) Personal() bool { return e == EditionPersonal }

// SaaS reports whether e is the saas edition.
func (e Edition) SaaS() bool { return e == EditionSaaS }

// String returns the value of APP_EDITION.
func (e Edition) String() string { return string(e) }

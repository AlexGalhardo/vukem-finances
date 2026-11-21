package config

import (
	"strings"

	vconfig "github.com/AlexGalhardo/vukem/config"
)

// AI is the host-level configuration of the AI layer (docs/ai.md).
type AI struct {
	// AllowPrivateHosts lets the openai_compatible provider reach private and
	// loopback addresses over http, for a local model server. Only the
	// personal edition honors it.
	AllowPrivateHosts bool
}

// LoadAI reads AI_ALLOW_PRIVATE_HOSTS from src. The saas edition ignores the
// variable: a tenant must never be able to aim the server at its own network.
func LoadAI(src vconfig.Lookuper, edition Edition) AI {
	v, _ := src.Lookup("AI_ALLOW_PRIVATE_HOSTS")

	return AI{AllowPrivateHosts: edition.Personal() && strings.EqualFold(strings.TrimSpace(v), "true")}
}

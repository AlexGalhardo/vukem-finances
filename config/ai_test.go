package config_test

import (
	"testing"

	appconfig "github.com/AlexGalhardo/vukem-finances/config"
)

func TestLoadAIPrivateHostsOnlyInPersonal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		edition appconfig.Edition
		env     mapSource
		want    bool
	}{
		{"personal unset", appconfig.EditionPersonal, mapSource{}, false},
		{"personal true", appconfig.EditionPersonal, mapSource{"AI_ALLOW_PRIVATE_HOSTS": "true"}, true},
		{"personal false", appconfig.EditionPersonal, mapSource{"AI_ALLOW_PRIVATE_HOSTS": "false"}, false},
		{"saas ignores true", appconfig.EditionSaaS, mapSource{"AI_ALLOW_PRIVATE_HOSTS": "true"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := appconfig.LoadAI(tc.env, tc.edition).AllowPrivateHosts; got != tc.want {
				t.Errorf("AllowPrivateHosts = %v, want %v", got, tc.want)
			}
		})
	}
}

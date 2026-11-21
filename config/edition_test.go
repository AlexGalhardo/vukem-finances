package config_test

import (
	"strings"
	"testing"

	appconfig "github.com/AlexGalhardo/vukem-finances/config"
)

type mapSource map[string]string

func (m mapSource) Lookup(k string) (string, bool) { v, ok := m[k]; return v, ok }

func TestLoadEdition(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		value   string
		set     bool
		want    appconfig.Edition
		wantErr bool
	}{
		{name: "unset", want: appconfig.EditionPersonal},
		{name: "empty", value: "  ", set: true, want: appconfig.EditionPersonal},
		{name: "personal", value: "personal", set: true, want: appconfig.EditionPersonal},
		{name: "saas", value: "saas", set: true, want: appconfig.EditionSaaS},
		{name: "wrong case", value: "Saas", set: true, wantErr: true},
		{name: "unknown", value: "bogus", set: true, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			env := mapSource{}
			if tc.set {
				env["APP_EDITION"] = tc.value
			}

			got, err := appconfig.LoadEdition(env)
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "personal") || !strings.Contains(err.Error(), "saas") {
					t.Fatalf("err = %v, want one naming both editions", err)
				}

				return
			}

			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

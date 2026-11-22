package config_test

import (
	"slices"
	"testing"

	appconfig "github.com/AlexGalhardo/vukem-finances/config"
)

func TestLoadMCP(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		env         map[string]string
		toolsets    []string
		destructive bool
		wantErr     bool
	}{
		{name: "defaults are read only", toolsets: []string{"read"}},
		{name: "blank toolsets keep the default", env: map[string]string{"MCP_TOOLSETS": "  "}, toolsets: []string{"read"}},
		{name: "a list", env: map[string]string{"MCP_TOOLSETS": "read, Transactions,read"}, toolsets: []string{"read", "transactions"}},
		{name: "all", env: map[string]string{"MCP_TOOLSETS": "all"}, toolsets: []string{"all"}},
		{name: "unknown toolset", env: map[string]string{"MCP_TOOLSETS": "read,write"}, wantErr: true},
		{name: "destructive on", env: map[string]string{"MCP_ALLOW_DESTRUCTIVE": "true"}, toolsets: []string{"read"}, destructive: true},
		{name: "destructive off", env: map[string]string{"MCP_ALLOW_DESTRUCTIVE": "false"}, toolsets: []string{"read"}},
		{name: "destructive typo", env: map[string]string{"MCP_ALLOW_DESTRUCTIVE": "yes"}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := appconfig.LoadMCP(mapSource(tc.env))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr {
				return
			}

			if !slices.Equal(got.Toolsets, tc.toolsets) || got.AllowDestructive != tc.destructive {
				t.Fatalf("got %+v, want toolsets %v destructive %v", got, tc.toolsets, tc.destructive)
			}
		})
	}
}

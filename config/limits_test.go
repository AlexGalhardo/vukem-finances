package config

import (
	"testing"

	vconfig "github.com/AlexGalhardo/vukem/config"
)

func TestWorkspaceReadLimit(t *testing.T) {
	t.Parallel()

	for value, want := range map[string]int{"": ReadLimit, " 500 ": 500, "1": 1} {
		got, err := WorkspaceReadLimit(vconfig.MapSource(map[string]string{"WORKSPACE_READ_LIMIT": value}))
		if err != nil || got != want {
			t.Errorf("WORKSPACE_READ_LIMIT=%q = %d, %v; want %d", value, got, err, want)
		}
	}

	for _, value := range []string{"0", "-1", "many", "1.5"} {
		if _, err := WorkspaceReadLimit(vconfig.MapSource(map[string]string{"WORKSPACE_READ_LIMIT": value})); err == nil {
			t.Errorf("WORKSPACE_READ_LIMIT=%q was accepted", value)
		}
	}
}

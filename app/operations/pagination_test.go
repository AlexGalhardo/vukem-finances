package operations_test

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/operations"
)

func TestCursorIsOpaqueAndStrict(t *testing.T) {
	t.Parallel()

	first, err := operations.ParseCursor("")
	if err != nil || first.Page != 1 {
		t.Fatalf("empty cursor = %+v, %v, want page 1", first, err)
	}

	c := operations.Cursor{Page: 7}
	if got := c.String(); got == "7" || got == "" {
		t.Errorf("cursor text %q is not opaque", got)
	}

	back, err := operations.ParseCursor(c.String())
	if err != nil || back != c {
		t.Fatalf("round trip = %+v, %v", back, err)
	}

	for _, bad := range []string{"7", "!!!", "cDE6MDA", "cDE6LTE", "cDE6YQ"} {
		if _, err := operations.ParseCursor(bad); !operations.IsCode(err, operations.CodeValidation) {
			t.Errorf("cursor %q: %v, want validation_failed", bad, err)
		}
	}
}

package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// Every color a category or tag may carry is a design token in both themes, so the swatch never renders empty
// (resources/js/design/contrast.test.ts measures their contrast; its list must match models.CategoryColors).
func TestCategoryColorsHaveTokens(t *testing.T) {
	t.Parallel()

	css, err := os.ReadFile("../resources/css/tokens.css")
	if err != nil {
		t.Fatal(err)
	}

	test, err := os.ReadFile("../resources/js/design/contrast.test.ts")
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range models.CategoryColors {
		if n := strings.Count(string(css), "--vf-cat-"+c+":"); n != 2 {
			t.Errorf("--vf-cat-%s is defined %d times in tokens.css, want once per theme", c, n)
		}

		if !strings.Contains(string(test), `"`+c+`"`) {
			t.Errorf("contrast.test.ts does not measure %s", c)
		}
	}
}

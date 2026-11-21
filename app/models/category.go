package models

import (
	"database/sql"
	"slices"

	"github.com/AlexGalhardo/vukem/orm"
)

// CategoryColors are the color tokens a category or tag may carry: names, not
// hex values. Each one has a --vf-cat-<name> token in resources/css/tokens.css
// (checked for contrast by resources/js/design/contrast.test.ts). The empty
// string means "no color".
var CategoryColors = []string{
	"slate", "blue", "teal", "green", "olive", "amber", "orange", "red", "pink", "violet",
}

// CategoryIcons are the icon names a category may carry; the page maps each to
// a lucide icon. The empty string means "no icon".
var CategoryIcons = []string{
	"wallet", "home", "cart", "car", "heart", "film", "utensils", "briefcase",
	"gift", "plane", "book", "paw", "zap", "coins", "shirt", "tag",
}

// ValidColor reports whether c is a palette name or empty.
func ValidColor(c string) bool { return c == "" || slices.Contains(CategoryColors, c) }

// ValidIcon reports whether i is an icon name or empty.
func ValidIcon(i string) bool { return i == "" || slices.Contains(CategoryIcons, i) }

// Category groups transactions of one kind ("income" or "expense"). It nests
// one level: a child has a ParentID, a parent has none, and both have the
// same kind.
type Category struct {
	orm.Model
	orm.SoftDeletes

	WorkspaceID int64  `json:"workspace_id"`
	UID         string `json:"uid"`
	Version     int64  `json:"version" orm:"readonly"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`

	ParentID sql.Null[int64] `json:"parent_id"`
	Color    string          `json:"color"`
	Icon     string          `json:"icon"`

	// ArchivedAt is set while the category is archived: hidden from the
	// pickers, history and budgets kept.
	ArchivedAt orm.NullTime `json:"archived_at"`
}

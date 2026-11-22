package operations

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
)

// Cursor is the opaque position of a page in a list. Every list operation that
// pages takes and gives it as text, so a surface never builds or reads one: it
// hands back what it was given. The text is an implementation detail and can
// change; a cursor kept across a release is not promised to work.
//
// The lists that existed before the registry (transactions.list pages with
// ?page=N) keep their public contract; none of the lists of epic 30 pages.
type Cursor struct {
	// Page is the 1-based page the cursor points at.
	Page int
}

const cursorPrefix = "p1:"

// String is the opaque text of the cursor.
func (c Cursor) String() string {
	return base64.RawURLEncoding.EncodeToString([]byte(cursorPrefix + strconv.Itoa(c.Page)))
}

// ParseCursor reads the text String made. The empty text is the first page; any
// other text that String did not make is a validation error.
func ParseCursor(text string) (Cursor, error) {
	if text == "" {
		return Cursor{Page: 1}, nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil || !strings.HasPrefix(string(raw), cursorPrefix) {
		return Cursor{}, errBadCursor
	}

	page, err := strconv.Atoi(strings.TrimPrefix(string(raw), cursorPrefix))
	if err != nil || page < 1 || strconv.Itoa(page) != strings.TrimPrefix(string(raw), cursorPrefix) {
		return Cursor{}, errBadCursor
	}

	return Cursor{Page: page}, nil
}

var errBadCursor = fail(http.StatusUnprocessableEntity, CodeValidation, "The cursor is not one this server gave.")

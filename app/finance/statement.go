package finance

import (
	"bytes"
	"mime"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/AlexGalhardo/vukem/errs"
)

// StatementRow is one line of a bank statement as a parser read it, before it
// becomes a transaction. Money is signed int64 cents: a negative amount is an
// expense. A row the parser could not read keeps its line number and the
// reason in Problem and carries no data.
type StatementRow struct {
	// Line is the 1-based line of the file the row starts on (for CSV, the
	// header is line 1).
	Line int
	// Date is an ISO date, "2026-11-12".
	Date        string
	Description string
	Cents       int64
	// Category is the category name the file holds for the row, if any; the
	// importer matches it to a category of the workspace.
	Category string
	// FITID is the bank's own id of the transaction (OFX); it makes the
	// duplicate check exact.
	FITID string
	// Problem is why the row cannot be imported; empty for a good row.
	Problem string
}

// StatementParser reads one statement format. Parse never trusts the content:
// it bounds what it reads, and a row it cannot use is a StatementRow with a
// Problem, not an error. An error means the file as a whole is unusable.
type StatementParser interface {
	Parse(content []byte) ([]StatementRow, error)
}

var (
	errImportType = errs.New(http.StatusUnprocessableEntity,
		"Upload a .csv, .ofx or .qfx file.")
	errImportMismatch = errs.New(http.StatusUnprocessableEntity,
		"The file name, its type and its content do not agree: a CSV must be named .csv and an OFX file .ofx or .qfx.")
	errImportBinary = errs.New(http.StatusUnprocessableEntity, "The file is not a text statement.")
)

// uploadTypes are the media types a statement may arrive with, by format. The
// generic ones (text/plain, application/octet-stream, nothing at all) fit both:
// browsers send them for files they do not know.
var uploadTypes = map[string][]string{
	FormatCSV: {"text/csv", "application/csv", "application/vnd.ms-excel", "text/comma-separated-values"},
	FormatOFX: {"application/x-ofx", "application/ofx", "text/x-ofx", "application/vnd.intu.qfx", "text/xml", "application/xml"},
}

// ErrImportTooBig is the answer to a file over MaxImportBytes.
var ErrImportTooBig = errImportTooBig

var genericTypes = []string{"", "text/plain", "application/octet-stream"}

// formatOfExtension maps a file name to a format; "" when the extension is not one of ours.
func formatOfExtension(filename string) string {
	switch strings.ToLower(path.Ext(filename)) {
	case ".csv":
		return FormatCSV
	case ".ofx", ".qfx":
		return FormatOFX
	}

	return ""
}

// CheckUpload checks the extension and the Content-Type of an upload against
// each other: both must name the same format, or the type must be a generic
// one. It does not look at the content; DetectFormat does.
func CheckUpload(filename, contentType string) error {
	format := formatOfExtension(filename)
	if format == "" {
		return errImportType
	}

	media, _, err := mime.ParseMediaType(contentType)
	if err != nil && contentType != "" {
		return errImportMismatch
	}

	media = strings.ToLower(media)

	if slices.Contains(genericTypes, media) || slices.Contains(uploadTypes[format], media) {
		return nil
	}

	return errImportMismatch
}

// looksLikeOFX sniffs the start of a file: an OFX 1.x header, or the <OFX> tag
// of either version.
func looksLikeOFX(content []byte) bool {
	head := bytes.TrimPrefix(content[:min(len(content), 4096)], []byte{0xEF, 0xBB, 0xBF})
	upper := bytes.ToUpper(head)

	return bytes.HasPrefix(bytes.TrimLeft(upper, " \t\r\n"), []byte("OFXHEADER:")) || bytes.Contains(upper, []byte("<OFX>"))
}

// DetectFormat names the format of an upload from its extension and its
// content. They must agree: a .csv that holds OFX, or an .ofx that does not,
// is refused rather than guessed at. A file with a NUL byte is not text.
func DetectFormat(filename string, content []byte) (string, error) {
	format := formatOfExtension(filename)
	if format == "" {
		return "", errImportType
	}

	if bytes.IndexByte(content, 0) >= 0 {
		return "", errImportBinary
	}

	if looksLikeOFX(content) != (format == FormatOFX) {
		if format == FormatOFX {
			return "", errImportOFX
		}

		return "", errImportMismatch
	}

	return format, nil
}

// SpreadsheetSafe makes a text safe to put in a cell of an exported
// spreadsheet. A cell that starts with =, +, - or @ (or a tab or carriage
// return, which some programs skip) is read as a formula, so one that came
// from a statement ("=HYPERLINK(...)") could run when the export is opened.
// A leading apostrophe makes it plain text. The importer stores text as typed,
// so every exporter must call this on each text cell it writes.
func SpreadsheetSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}

	return s
}

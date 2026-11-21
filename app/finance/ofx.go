package finance

import (
	"bytes"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem/errs"
)

// Limits of an OFX file, on top of MaxImportBytes and maxImportRows.
const (
	// maxOFXTagLength bounds "<NAME>": a longer one is not a tag.
	maxOFXTagLength = 64
	// maxOFXValueLength bounds a field value, in bytes.
	maxOFXValueLength = 512
)

var (
	errImportOFX = errs.New(http.StatusUnprocessableEntity,
		"The file is not a readable OFX statement: no bank transactions were found.")
	errOFXHostile = errs.New(http.StatusUnprocessableEntity,
		"The file declares a DTD, an entity or a CDATA section, which an OFX statement never needs, so it was refused.")
	errOFXMalformed = errs.New(http.StatusUnprocessableEntity, "The file is not well formed OFX.")
	errImportRows   = errs.New(http.StatusUnprocessableEntity, "The file has more than 20000 rows: split it and import the parts.")
)

// OFXParser reads an OFX statement, version 1.x (SGML, leaf tags without a
// closing tag) or 2.x (XML). It is a bounded scanner, not an XML decoder: it
// never resolves entities, never fetches a DTD or any other resource, and
// refuses a file that declares them (billion laughs and external entities
// need a declaration). Memory is the file itself (2 MiB at most) plus the
// rows, which stop at 20000.
type OFXParser struct{}

// Parse implements StatementParser.
func (OFXParser) Parse(content []byte) ([]StatementRow, error) {
	if len(content) > MaxImportBytes {
		return nil, errImportTooBig
	}

	if bytes.Contains(content, []byte("<!")) {
		return nil, errOFXHostile
	}

	text := decodeOFXText(content)

	var (
		rows []StatementRow
		cur  *ofxTxn
		leaf string
	)

	flush := func() error {
		if cur == nil {
			return nil
		}

		if len(rows) >= maxImportRows {
			return errImportRows
		}

		rows = append(rows, cur.row())
		cur = nil

		return nil
	}

	line := 1
	pos := 0

	for pos < len(text) {
		lt := strings.IndexByte(text[pos:], '<')
		if lt < 0 {
			break
		}

		// The text before this tag is the value of the leaf tag before it.
		value := text[pos : pos+lt]
		line += strings.Count(value, "\n")

		if leaf != "" && cur != nil {
			cur.set(leaf, value)
		}

		leaf = ""
		pos += lt

		if strings.HasPrefix(text[pos:], "<?") {
			// A processing instruction (<?xml ...?>, <?OFX ...?>) carries no data.
			end := strings.Index(text[pos:], "?>")
			if end < 0 {
				return nil, errOFXMalformed
			}

			pos += end + 2

			continue
		}

		end := strings.IndexByte(text[pos:min(len(text), pos+maxOFXTagLength+2)], '>')
		if end < 0 {
			return nil, errOFXMalformed
		}

		tag := text[pos+1 : pos+end]
		pos += end + 1

		switch {
		case strings.HasSuffix(tag, "/"):
			continue
		case strings.HasPrefix(tag, "/"):
			if strings.EqualFold(strings.TrimSpace(tag[1:]), "STMTTRN") {
				if err := flush(); err != nil {
					return nil, err
				}
			}
		default:
			name := strings.ToUpper(tagName(tag))
			if name == "" {
				return nil, errOFXMalformed
			}

			if name == "STMTTRN" {
				if err := flush(); err != nil {
					return nil, err
				}

				cur = &ofxTxn{line: line}
			}

			leaf = name
		}
	}

	if err := flush(); err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, errImportOFX
	}

	return rows, nil
}

func tagName(tag string) string {
	if i := strings.IndexAny(tag, " \t\r\n"); i >= 0 {
		tag = tag[:i]
	}

	return tag
}

// ofxTxn collects the fields of one STMTTRN.
type ofxTxn struct {
	line                         int
	date, amount, name, memo, id string
	tooLong                      bool
}

func (t *ofxTxn) set(field, raw string) {
	if len(raw) > maxOFXValueLength {
		t.tooLong = true
		return
	}

	v := strings.TrimSpace(decodeEntities(raw))

	switch field {
	case "DTPOSTED":
		t.date = v
	case "TRNAMT":
		t.amount = v
	case "NAME":
		t.name = v
	case "MEMO":
		t.memo = v
	case "FITID":
		t.id = v
	}
}

var (
	ofxDate = regexp.MustCompile(`^(\d{8})(\d{6})?(\.\d{1,3})?(\[[^\]]{0,16}\])?$`)
	// ofxAmount is a plain decimal with at most two places, with or without
	// thousands groups of three: the spec says "1234.56", and a few banks
	// write "1.234,56" or "1,234.56". Letters, signs inside, spaces and
	// currency symbols are not accepted (ParseAmount would drop them).
	ofxAmount = regexp.MustCompile(`^[+-]?(\d{1,3}([.,]\d{3})+([.,]\d{1,2})?|\d+([.,]\d{1,2})?|[.,]\d{1,2})$`)
)

func (t *ofxTxn) row() StatementRow {
	r := StatementRow{Line: t.line, FITID: t.id}

	if t.tooLong {
		r.Problem = "field too long"
		return r
	}

	m := ofxDate.FindStringSubmatch(t.date)
	if m == nil {
		r.Problem = "bad date"
		return r
	}

	day := m[1][:4] + "-" + m[1][4:6] + "-" + m[1][6:8]
	if !validDate(day) {
		r.Problem = "bad date"
		return r
	}

	// ParseAmount drops letters, so "1.5e3" would read as 1.53: only a plain
	// decimal with at most two places goes in.
	if !ofxAmount.MatchString(t.amount) {
		r.Problem = "bad amount"
		return r
	}

	cents, err := ParseAmount(t.amount)
	if err != nil || cents == 0 {
		r.Problem = "bad amount"
		return r
	}

	r.Date, r.Cents, r.Description = day, cents, t.name

	if r.Description == "" {
		r.Description = t.memo
	}

	return r
}

// decodeOFXText turns the bytes of a file into UTF-8: a BOM is dropped, valid
// UTF-8 is kept, and anything else is Windows-1252, the default charset of an
// OFX 1.x file (ISO-8859-1 text decodes the same, bar the 0x80-0x9f range).
func decodeOFXText(b []byte) string {
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})

	if utf8.Valid(b) {
		return string(b)
	}

	var sb strings.Builder

	sb.Grow(len(b) + len(b)/4)

	for _, c := range b {
		switch {
		case c < 0x80:
			sb.WriteByte(c)
		case c >= 0x80 && c <= 0x9f && cp1252[c-0x80] != 0:
			sb.WriteRune(cp1252[c-0x80])
		default:
			sb.WriteRune(rune(c))
		}
	}

	return sb.String()
}

// cp1252 maps 0x80-0x9f; zero means "the same code point" (undefined in 1252).
var cp1252 = [32]rune{
	0x20AC, 0, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021, 0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0, 0x017D, 0,
	0, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014, 0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0, 0x017E, 0x0178,
}

var predefined = strings.NewReplacer("&lt;", "<", "&gt;", ">", "&quot;", `"`, "&apos;", "'")

// decodeEntities resolves the five predefined entities and numeric character
// references; any other "&name;" stays as typed (nothing is ever looked up).
func decodeEntities(s string) string {
	if !strings.Contains(s, "&") {
		return s
	}

	var sb strings.Builder

	for len(s) > 0 {
		i := strings.IndexByte(s, '&')
		if i < 0 {
			sb.WriteString(s)
			break
		}

		sb.WriteString(s[:i])
		s = s[i:]

		semi := strings.IndexByte(s, ';')
		if semi < 0 || semi > 10 {
			sb.WriteByte('&')

			s = s[1:]

			continue
		}

		ent := s[:semi+1]
		s = s[semi+1:]

		switch {
		case strings.HasPrefix(ent, "&#"):
			sb.WriteString(numericEntity(ent))
		case ent == "&amp;":
			sb.WriteByte('&')
		default:
			sb.WriteString(predefined.Replace(ent))
		}
	}

	return sb.String()
}

func numericEntity(ent string) string {
	body := ent[2 : len(ent)-1]
	base := 10

	if strings.HasPrefix(body, "x") || strings.HasPrefix(body, "X") {
		body, base = body[1:], 16
	}

	n, err := strconv.ParseInt(body, base, 32)
	if err != nil || n <= 0 || n == 0xFFFD || !utf8.ValidRune(rune(n)) || n < 0x20 && n != '\t' && n != '\n' && n != '\r' {
		return ent
	}

	return string(rune(n))
}

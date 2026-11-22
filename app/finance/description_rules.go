package finance

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"unicode"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

const (
	// ruleTokens is how many significant words of a description make its key.
	ruleTokens = 4
	// ruleKeyMax bounds a key, in bytes (the column is 120 wide).
	ruleKeyMax = 120
	// longIDLength is the length from which a word that holds a digit is an
	// identifier (a card tail, a trace code) and not part of a store name.
	longIDLength = 6
)

// ErrNotUncategorized answers a suggestion for a transaction that already has
// a category, or is gone.
var ErrNotUncategorized = errs.New(http.StatusConflict, "That transaction is already categorized or no longer exists.")

// foldTable maps the accented letters of the Latin alphabets the app is used
// in to their plain letter. The standard library has no accent folding and the
// project adds no dependency for it.
var foldTable = map[rune]string{ //nolint:gochecknoglobals // read-only table
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
	'ç': "c", 'ć': "c", 'č': "c",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ę': "e", 'ě': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ī': "i", 'ı': "i",
	'ñ': "n", 'ń': "n", 'ň': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o", 'ő': "o",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ū': "u", 'ů': "u", 'ű': "u",
	'ý': "y", 'ÿ': "y", 'ś': "s", 'š': "s", 'ş': "s", 'ž': "z", 'ź': "z", 'ż': "z",
	'ł': "l", 'ð': "d", 'đ': "d", 'ß': "ss", 'æ': "ae", 'œ': "oe", 'þ': "th",
}

// noiseWords are the generic words of a bank line (and the small words of
// English, Portuguese and Spanish) that say nothing about what was bought.
var noiseWords = map[string]bool{ //nolint:gochecknoglobals // read-only table
	"de": true, "da": true, "do": true, "das": true, "dos": true, "del": true, "la": true, "el": true,
	"the": true, "of": true, "at": true, "and": true, "for": true, "to": true, "em": true, "no": true,
	"na": true, "para": true, "por": true, "con": true, "und": true,
	"compra": true, "compras": true, "purchase": true, "pagamento": true, "payment": true, "pgto": true,
	"pag": true, "cartao": true, "card": true, "pos": true, "ref": true, "debito": true, "pagto": true,
}

func fold(r rune) string {
	r = unicode.ToLower(r)
	if f, ok := foldTable[r]; ok {
		return f
	}

	return string(r)
}

// NormalizeDescription is the key of the memory of descriptions: lower case,
// accents folded, punctuation, dates, store numbers and long identifiers
// removed, whitespace collapsed, and the first four significant words. Two
// lines of the same shop ("STARBUCKS #1234 SP 12/11" and "Starbucks 99")
// have the same key. The empty string means nothing to remember.
func NormalizeDescription(description string) string {
	var tokens []string

	var word strings.Builder

	digits := false

	flush := func() {
		w := word.String()

		hadDigits := digits
		word.Reset()

		digits = false

		if w == "" {
			return
		}

		if hadDigits {
			if len(w) >= longIDLength {
				return
			}

			w = strings.Map(func(r rune) rune {
				if r >= '0' && r <= '9' {
					return -1
				}

				return r
			}, w)
		}

		if len(w) >= 2 && !noiseWords[w] && len(tokens) < ruleTokens {
			tokens = append(tokens, w)
		}
	}

	for _, r := range stripDates(description) {
		switch {
		case r >= '0' && r <= '9':
			word.WriteRune(r)

			digits = true
		case unicode.IsLetter(r):
			word.WriteString(fold(r))
		default:
			flush()
		}
	}

	flush()

	key := strings.Join(tokens, " ")
	if len(key) > ruleKeyMax {
		key = key[:ruleKeyMax]
	}

	return key
}

// stripDates blanks the numeric dates of a line (12/11, 12-11-2026, 2026.11.12):
// they change every day and would give every line its own key.
func stripDates(s string) string {
	b := []rune(s)
	out := make([]rune, 0, len(b))

	for i := 0; i < len(b); {
		if n := dateLength(b[i:]); n > 0 {
			out = append(out, ' ')
			i += n

			continue
		}

		out = append(out, b[i])
		i++
	}

	return string(out)
}

// dateLength returns how many runes at the start of r form a numeric date made
// of two or three numbers split by / . or -, and 0 when none does.
func dateLength(r []rune) int {
	if len(r) == 0 || r[0] < '0' || r[0] > '9' {
		return 0
	}

	i, groups := 0, 0

	for {
		start := i
		for i < len(r) && r[i] >= '0' && r[i] <= '9' {
			i++
		}

		if i == start || i-start > 4 {
			return 0
		}

		groups++

		if i+1 < len(r) && (r[i] == '/' || r[i] == '.' || r[i] == '-') && r[i+1] >= '0' && r[i+1] <= '9' && groups < 3 {
			i++

			continue
		}

		break
	}

	// A date has a separator: "12/11" or "2026-11-12", never a bare "5".
	if groups < 2 {
		return 0
	}

	if i < len(r) && unicode.IsLetter(r[i]) {
		return 0
	}

	return i
}

// LearnRule remembers that a description is of a category: every manual
// categorization and every accepted suggestion calls it. The same category
// again counts a hit; another category replaces the rule (the last correction
// wins). ex is the database or the transaction of the write that taught it.
func (s *Service) LearnRule(ctx context.Context, ex database.Executor, description string, categoryID int64, source string) error {
	key := NormalizeDescription(description)
	if key == "" || categoryID <= 0 {
		return nil
	}

	rule, err := workspaces.Scoped[models.DescriptionRule](ctx, ex).Where("key", key).First(ctx)

	switch {
	case err == nil:
		set := orm.Set{"hits": rule.Hits + 1}

		if rule.CategoryID != categoryID {
			set = orm.Set{"category_id": categoryID, "hits": 1, "source": source}
		} else if source == models.RuleSourceUser {
			set["source"] = source
		}

		_, err = workspaces.Scoped[models.DescriptionRule](ctx, ex).Where("id", rule.ID).Update(ctx, set)

		return err
	case errors.Is(err, orm.ErrNotFound):
		r := models.DescriptionRule{Key: key, CategoryID: categoryID, Hits: 1, Source: source}
		if err := orm.Create(ctx, ex, &r); err != nil && !s.DB.UniqueViolation(err) {
			return err
		}

		return nil
	default:
		return err
	}
}

// RuleCategory is the category the workspace chose for this description of a
// transaction of kind, when one is remembered and still usable: it exists, is
// not archived and is of the same kind. It uses no AI and sends nothing out.
func (s *Service) RuleCategory(ctx context.Context, ex database.Executor, description, kind string) (models.Category, bool, error) {
	key := NormalizeDescription(description)
	if key == "" {
		return models.Category{}, false, nil
	}

	rule, err := workspaces.Scoped[models.DescriptionRule](ctx, ex).Where("key", key).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return models.Category{}, false, nil
	}

	if err != nil {
		return models.Category{}, false, err
	}

	cat, err := workspaces.Scoped[models.Category](ctx, ex).Where("id", rule.CategoryID).
		Where("kind", kind).WhereNull("archived_at").First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return models.Category{}, false, nil
	}

	return cat, err == nil, err
}

// AssignCategory gives an uncategorized transaction its category: the one write
// of a suggestion. It refuses a transaction that is gone, has a category or is
// a transfer leg (ErrNotUncategorized), and a category that is gone, archived
// or of another kind (the category error). The update is conditional, so two
// concurrent accepts change the row once. Run it inside the workspace lock.
func (s *Service) AssignCategory(ctx context.Context, tx *database.Tx, txUID, categoryUID string) (models.Transaction, error) {
	row, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("uid", txUID).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return models.Transaction{}, ErrNotUncategorized
	}

	if err != nil {
		return models.Transaction{}, err
	}

	if row.CategoryID.Valid || row.TransferUID.Valid {
		return models.Transaction{}, ErrNotUncategorized
	}

	cat, err := workspaces.Scoped[models.Category](ctx, tx).Where("uid", categoryUID).
		Where("kind", row.Kind).WhereNull("archived_at").First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return models.Transaction{}, errCategory
	}

	if err != nil {
		return models.Transaction{}, err
	}

	n, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("id", row.ID).WhereNull("category_id").
		WhereNull("transfer_uid").Update(ctx, orm.Set{"category_id": cat.ID})
	if err != nil {
		return models.Transaction{}, err
	}

	if n == 0 {
		return models.Transaction{}, ErrNotUncategorized
	}

	row.CategoryID = sql.Null[int64]{V: cat.ID, Valid: true}

	return row, nil
}

// ClearCategory takes the category back from a transaction, only while it is
// still the one a suggestion applied: a category the person changed since is
// theirs and stays. It reports whether the row changed.
func (s *Service) ClearCategory(ctx context.Context, tx *database.Tx, txUID, categoryUID string) (bool, error) {
	cat, err := workspaces.Scoped[models.Category](ctx, tx).Where("uid", categoryUID).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	n, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("uid", txUID).Where("category_id", cat.ID).
		Update(ctx, orm.Set{"category_id": nil})

	return n > 0, err
}

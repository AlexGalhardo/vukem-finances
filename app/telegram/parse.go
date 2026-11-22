package telegram

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// The ways a message can fail to be an entry.
var (
	ErrEmpty     = errors.New("telegram: empty message")
	ErrTooLong   = errors.New("telegram: message is too long")
	ErrNoAmount  = errors.New("telegram: no amount")
	ErrBadAmount = errors.New("telegram: bad amount")
	ErrBadDate   = errors.New("telegram: bad date")
)

// Entry is a transaction read from a message, before it is confirmed.
type Entry struct {
	Kind        string
	Cents       int64
	Description string
	// Date is an ISO date.
	Date string
}

// amountToken is the shape of the amount word: digits and separators, with an
// optional currency symbol in front. The value itself is read by
// finance.ParseAmount, the only decimal parser of the app.
var amountToken = regexp.MustCompile(`^(?:R\$|\$|€|£)?[0-9][0-9.,]*$`)

var dayMonth = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})(?:/(\d{4}))?$`)

var isoDateWord = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ParseEntry reads "[+|-]amount [description] [date]". A leading + is income,
// a leading - or nothing is an expense. The date word at the end is today,
// yesterday, hoje, ontem, hoy, ayer, dd/mm, dd/mm/yyyy or an ISO date; without
// one the entry is for today. today is the current date in the workspace
// timezone, as a UTC midnight. Input beyond config.TelegramInputMax characters is refused.
func ParseEntry(text string, today time.Time) (Entry, error) {
	if utf8.RuneCountInString(text) > config.TelegramInputMax {
		return Entry{}, ErrTooLong
	}

	// A control character has no place in a description.
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}

		return r
	}, text)

	words := strings.Fields(text)
	if len(words) == 0 {
		return Entry{}, ErrEmpty
	}

	e := Entry{Kind: models.KindExpense}

	first := words[0]

	switch first[0] {
	case '+':
		e.Kind, first = models.KindIncome, first[1:]
	case '-':
		first = first[1:]
	}

	if first == "" {
		// "+ 3000 salary": the sign stands alone.
		if len(words) < 2 {
			return Entry{}, ErrNoAmount
		}

		words, first = words[1:], words[1]
	}

	if !amountToken.MatchString(first) {
		return Entry{}, ErrNoAmount
	}

	cents, err := finance.ParseAmount(first)
	if err != nil || cents <= 0 {
		return Entry{}, ErrBadAmount
	}

	e.Cents = cents
	rest := words[1:]
	e.Date = today.Format("2006-01-02")

	if n := len(rest); n > 0 {
		date, ok, err := dateWord(rest[n-1], today)
		if err != nil {
			return Entry{}, err
		}

		if ok {
			e.Date, rest = date, rest[:n-1]
		}
	}

	e.Description = strings.Join(rest, " ")

	return e, nil
}

// dateWord reads w as a date. ok is false when w is not a date word at all
// (it is then part of the description); err is set for a date word that names
// no real day.
func dateWord(w string, today time.Time) (string, bool, error) {
	switch strings.ToLower(w) {
	case "today", "hoje", "hoy":
		return today.Format("2006-01-02"), true, nil
	case "yesterday", "ontem", "ayer":
		return today.Add(-24 * time.Hour).Format("2006-01-02"), true, nil
	}

	if isoDateWord.MatchString(w) {
		d, err := time.Parse("2006-01-02", w)
		if err != nil {
			return "", true, ErrBadDate
		}

		return d.Format("2006-01-02"), true, nil
	}

	m := dayMonth.FindStringSubmatch(w)
	if m == nil {
		return "", false, nil
	}

	year := today.Year()
	if m[3] != "" {
		year, _ = strconv.Atoi(m[3])
	}

	day, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])

	d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	// time.Date normalizes 31/02 into March: a real date round-trips.
	if d.Day() != day || int(d.Month()) != month || month < 1 || month > 12 {
		return "", true, ErrBadDate
	}

	return d.Format("2006-01-02"), true, nil
}

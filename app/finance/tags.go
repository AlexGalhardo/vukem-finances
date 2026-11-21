package finance

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var (
	errTagName = errs.New(http.StatusUnprocessableEntity,
		"A tag name is 1 to 40 characters, on one line.")
	errTagsPerTx = errs.New(http.StatusUnprocessableEntity,
		"A transaction takes at most 10 tags.")
	errTagsPerWorkspace = errs.New(http.StatusUnprocessableEntity,
		"A workspace holds at most 200 tags: delete one you no longer use.")
	errTagUnknown = errs.New(http.StatusUnprocessableEntity, "Choose one of your tags.")
	errTagTaken   = errs.New(http.StatusConflict, "A tag with that name already exists.")
)

// TagRef is a tag as a transaction row shows it.
type TagRef struct {
	ID    int64  `json:"id"`
	UID   string `json:"uid"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// cleanTagName trims a tag name and checks its length and characters.
func cleanTagName(name string) (string, error) {
	name = strings.TrimSpace(name)

	n := utf8.RuneCountInString(name)
	if n == 0 || n > models.MaxTagNameLength || !utf8.ValidString(name) {
		return "", errTagName
	}

	for _, r := range name {
		if unicode.IsControl(r) {
			return "", errTagName
		}
	}

	return name, nil
}

// Tags lists the live tags of the workspace of ctx by name.
func (s *Service) Tags(ctx context.Context) ([]models.Tag, error) {
	return workspaces.Scoped[models.Tag](ctx, s.DB).OrderBy("name", orm.Asc).Get(ctx)
}

func tagByName(ctx context.Context, ex database.Executor, name string) (models.Tag, bool, error) {
	t, err := workspaces.Scoped[models.Tag](ctx, ex).WhereRaw("lower(name) = lower(?)", name).First(ctx)
	if notFound(err) {
		return models.Tag{}, false, nil
	}

	return t, err == nil, err
}

// createTag adds a tag inside tx, which holds the workspace lock. The names
// are unique without regard to case, and a workspace holds at most 200.
func (s *Service) createTag(ctx context.Context, tx *database.Tx, name, color string) (models.Tag, error) {
	if !models.ValidColor(color) {
		return models.Tag{}, errColor
	}

	n, err := workspaces.Scoped[models.Tag](ctx, tx).Count(ctx)
	if err != nil {
		return models.Tag{}, err
	}

	if n >= models.MaxTagsPerWorkspace {
		return models.Tag{}, errTagsPerWorkspace
	}

	t := models.Tag{Name: name, Color: color}
	if err := orm.Create(ctx, tx, &t); err != nil {
		if s.DB.UniqueViolation(err) {
			return models.Tag{}, errTagTaken
		}

		return models.Tag{}, err
	}

	return reload[models.Tag](ctx, tx, t.ID)
}

// CreateTag adds a tag.
func (s *Service) CreateTag(ctx context.Context, name, color string) (models.Tag, error) {
	name, err := cleanTagName(name)
	if err != nil {
		return models.Tag{}, err
	}

	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Tag{}, err
	}

	var out models.Tag

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		if _, found, err := tagByName(ctx, tx, name); err != nil || found {
			return firstErr(err, errTagTaken)
		}

		out, err = s.createTag(ctx, tx, name, color)

		return err
	})

	return out, err
}

// UpdateTag renames and recolors a tag. Transactions keep it: they point at
// its uid, not at its name.
func (s *Service) UpdateTag(ctx context.Context, id int64, name, color string) (models.Tag, error) {
	name, err := cleanTagName(name)
	if err != nil {
		return models.Tag{}, err
	}

	if !models.ValidColor(color) {
		return models.Tag{}, errColor
	}

	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Tag{}, err
	}

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		if _, err := reload[models.Tag](ctx, tx, id); err != nil {
			return err
		}

		if other, found, err := tagByName(ctx, tx, name); err != nil || (found && other.ID != id) {
			return firstErr(err, errTagTaken)
		}

		_, err := workspaces.Scoped[models.Tag](ctx, tx).Where("id", id).Update(ctx, orm.Set{"name": name, "color": color})
		if s.DB.UniqueViolation(err) {
			return errTagTaken
		}

		return err
	})
	if err != nil {
		return models.Tag{}, err
	}

	return reload[models.Tag](ctx, s.DB, id)
}

// DeleteTag deletes a tag and takes it off every transaction that carried it,
// each change logged.
func (s *Service) DeleteTag(ctx context.Context, id int64) error {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return err
	}

	at := s.now().UTC()

	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		if err := softDelete[models.Tag](ctx, tx, id, at); err != nil {
			return err
		}

		txIDs, err := workspaces.Scoped[models.TransactionTag](ctx, tx).Where("tag_id", id).
			Pluck[int64](ctx, "transaction_id")
		if err != nil {
			return err
		}

		if _, err := workspaces.Scoped[models.TransactionTag](ctx, tx).Where("tag_id", id).Delete(ctx); err != nil {
			return err
		}

		for _, txID := range txIDs {
			if err := syncTagUIDs(ctx, tx, txID); err != nil {
				return err
			}
		}

		return nil
	})
}

// tagIDsOf lists the tags attached to a transaction.
func tagIDsOf(ctx context.Context, ex database.Executor, transactionID int64) ([]int64, error) {
	return workspaces.Scoped[models.TransactionTag](ctx, ex).Where("transaction_id", transactionID).
		Pluck[int64](ctx, "tag_id")
}

// syncTagUIDs rewrites tag_uids of a transaction from its join rows, so the
// synced field and the table never disagree.
func syncTagUIDs(ctx context.Context, tx *database.Tx, transactionID int64) error {
	ids, err := tagIDsOf(ctx, tx, transactionID)
	if err != nil {
		return err
	}

	uids := []string{}

	if len(ids) > 0 {
		if uids, err = workspaces.Scoped[models.Tag](ctx, tx).WhereIn("id", anys(ids)...).Pluck[string](ctx, "uid"); err != nil {
			return err
		}
	}

	slices.Sort(uids)

	raw, err := json.Marshal(uids)
	if err != nil {
		return err
	}

	if _, err = workspaces.Scoped[models.Transaction](ctx, tx).Where("id", transactionID).
		Update(ctx, orm.Set{"tag_uids": string(raw)}); err != nil {
		return err
	}

	// A transaction in the trash can be restored, so it is kept in step too.
	_, err = workspaces.Scoped[models.Transaction](ctx, tx).OnlyTrashed().Where("id", transactionID).
		Update(ctx, orm.Set{"tag_uids": string(raw)})

	return err
}

// requireTaggable loads a live transaction that may carry tags: a transfer leg
// may not (its rules are those of the transfer, not of a label).
func requireTaggable(ctx context.Context, ex database.Executor, transactionID int64) error {
	t, err := workspaces.Scoped[models.Transaction](ctx, ex).Where("id", transactionID).First(ctx)
	if err != nil {
		return err
	}

	if t.TransferUID.Valid {
		return errIsTransfer
	}

	return nil
}

// replaceTags makes names the whole tag set of a transaction inside tx, which
// holds the workspace lock: tags that do not exist yet are created, and the
// limits apply to the result. Nothing changes if any name is refused.
func (s *Service) replaceTags(ctx context.Context, tx *database.Tx, transactionID int64, names []string) error {
	var wanted []string

	seen := map[string]bool{}

	for _, raw := range names {
		name, err := cleanTagName(raw)
		if err != nil {
			return err
		}

		if key := strings.ToLower(name); !seen[key] {
			seen[key] = true

			wanted = append(wanted, name)
		}
	}

	if len(wanted) > models.MaxTagsPerTransaction {
		return errTagsPerTx
	}

	ids := make([]int64, 0, len(wanted))

	for _, name := range wanted {
		t, found, err := tagByName(ctx, tx, name)
		if err != nil {
			return err
		}

		if !found {
			if t, err = s.createTag(ctx, tx, name, ""); err != nil {
				return err
			}
		}

		ids = append(ids, t.ID)
	}

	return s.writeJoin(ctx, tx, transactionID, ids)
}

func (s *Service) writeJoin(ctx context.Context, tx *database.Tx, transactionID int64, tagIDs []int64) error {
	if _, err := workspaces.Scoped[models.TransactionTag](ctx, tx).Where("transaction_id", transactionID).Delete(ctx); err != nil {
		return err
	}

	for _, id := range tagIDs {
		if err := orm.Create(ctx, tx, &models.TransactionTag{TransactionID: transactionID, TagID: id}); err != nil {
			return err
		}
	}

	return syncTagUIDs(ctx, tx, transactionID)
}

// AttachTag puts a tag of the workspace on a transaction of the workspace;
// attaching it twice changes nothing. A transaction takes at most 10 tags.
func (s *Service) AttachTag(ctx context.Context, transactionID, tagID int64) error {
	return s.changeTag(ctx, transactionID, tagID, true)
}

// DetachTag takes a tag off a transaction.
func (s *Service) DetachTag(ctx context.Context, transactionID, tagID int64) error {
	return s.changeTag(ctx, transactionID, tagID, false)
}

func (s *Service) changeTag(ctx context.Context, transactionID, tagID int64, attach bool) error {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return err
	}

	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		if err := requireTaggable(ctx, tx, transactionID); err != nil {
			return err
		}

		if _, err := reload[models.Tag](ctx, tx, tagID); err != nil {
			return orElse(err, errTagUnknown)
		}

		ids, err := tagIDsOf(ctx, tx, transactionID)
		if err != nil {
			return err
		}

		has := slices.Contains(ids, tagID)

		switch {
		case attach && has, !attach && !has:
			return nil
		case attach:
			if len(ids) >= models.MaxTagsPerTransaction {
				return errTagsPerTx
			}

			ids = append(ids, tagID)
		default:
			ids = slices.DeleteFunc(ids, func(id int64) bool { return id == tagID })
		}

		return s.writeJoin(ctx, tx, transactionID, ids)
	})
}

// tagsOf maps each transaction id to its tags, by name.
func (s *Service) tagsOf(ctx context.Context, transactionIDs []int64) (map[int64][]TagRef, error) {
	out := map[int64][]TagRef{}
	if len(transactionIDs) == 0 {
		return out, nil
	}

	links, err := workspaces.Scoped[models.TransactionTag](ctx, s.DB).WhereIn("transaction_id", anys(transactionIDs)...).Get(ctx)
	if err != nil || len(links) == 0 {
		return out, err
	}

	tagIDs := make([]int64, 0, len(links))
	for _, l := range links {
		tagIDs = append(tagIDs, l.TagID)
	}

	tags, err := workspaces.Scoped[models.Tag](ctx, s.DB).WhereIn("id", anys(tagIDs)...).Get(ctx)
	if err != nil {
		return nil, err
	}

	byID := map[int64]models.Tag{}
	for _, t := range tags {
		byID[t.ID] = t
	}

	for _, l := range links {
		if t, ok := byID[l.TagID]; ok {
			out[l.TransactionID] = append(out[l.TransactionID], TagRef{ID: t.ID, UID: t.UID, Name: t.Name, Color: t.Color})
		}
	}

	for id := range out {
		slices.SortFunc(out[id], func(a, b TagRef) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	}

	return out, nil
}

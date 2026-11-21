package finance

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var (
	errCategoryParent = errs.New(http.StatusUnprocessableEntity,
		"A subcategory goes under a top-level category of the same kind.")
	errCategoryNested = errs.New(http.StatusUnprocessableEntity,
		"A category with subcategories cannot become a subcategory.")
	errCategoryChildren = errs.New(http.StatusUnprocessableEntity,
		"Move or delete its subcategories first.")
	errReassign = errs.New(http.StatusUnprocessableEntity,
		"Choose another category of the same kind to receive its transactions and budget.")
	errColor          = errs.New(http.StatusUnprocessableEntity, "Choose a color from the list.")
	errIcon           = errs.New(http.StatusUnprocessableEntity, "Choose an icon from the list.")
	errParentArchived = errs.New(http.StatusConflict, "Unarchive the parent category first.")
)

// CategoryInput is a category as the form sends it. Kind matters on creation
// only: a category never changes kind, and a subcategory takes its parent's.
type CategoryInput struct {
	Name     string
	Kind     string
	ParentID int64
	Color    string
	Icon     string
}

// ActiveCategories lists the categories a picker offers: the archived ones are left out.
func (s *Service) ActiveCategories(ctx context.Context) ([]models.Category, error) {
	return workspaces.Scoped[models.Category](ctx, s.DB).WhereNull("archived_at").
		OrderBy("kind", orm.Asc).OrderBy("name", orm.Asc).Get(ctx)
}

func checkLook(color, icon string) error {
	if !models.ValidColor(color) {
		return errColor
	}

	if !models.ValidIcon(icon) {
		return errIcon
	}

	return nil
}

func notFound(err error) bool {
	return errors.Is(err, orm.ErrNotFound) || errors.Is(err, errs.ErrNotFound)
}

// orElse returns err unless it only says "no such row", in which case it
// returns fallback: a foreign or missing id is the same answer as a bad one.
func orElse(err, fallback error) error {
	if notFound(err) {
		return fallback
	}

	return err
}

// parentFor checks that parentID may be the parent of a category of kind that
// has children or not: one level only, same kind, live and not archived.
func parentFor(
	ctx context.Context, ex database.Executor, parentID, self int64, kind string, hasChildren bool,
) (sql.Null[int64], error) {
	if parentID == 0 {
		return sql.Null[int64]{}, nil
	}

	if parentID == self {
		return sql.Null[int64]{}, errCategoryParent
	}

	p, err := workspaces.Scoped[models.Category](ctx, ex).Where("id", parentID).First(ctx)
	if err != nil {
		return sql.Null[int64]{}, orElse(err, errCategoryParent)
	}

	if p.ParentID.Valid || p.Kind != kind || p.ArchivedAt.Valid {
		return sql.Null[int64]{}, errCategoryParent
	}

	if hasChildren {
		return sql.Null[int64]{}, errCategoryNested
	}

	return sql.Null[int64]{V: parentID, Valid: true}, nil
}

// AddCategory adds a category, top-level or under a parent. Names are unique
// per workspace.
func (s *Service) AddCategory(ctx context.Context, in CategoryInput) (models.Category, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return models.Category{}, errName
	}

	if err := checkLook(in.Color, in.Icon); err != nil {
		return models.Category{}, err
	}

	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Category{}, err
	}

	var id int64

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		kind := in.Kind

		if in.ParentID > 0 && kind == "" {
			p, err := workspaces.Scoped[models.Category](ctx, tx).Where("id", in.ParentID).First(ctx)
			if err != nil {
				return orElse(err, errCategoryParent)
			}

			kind = p.Kind
		}

		if kind != models.KindIncome && kind != models.KindExpense {
			return errKind
		}

		parent, err := parentFor(ctx, tx, in.ParentID, 0, kind, false)
		if err != nil {
			return err
		}

		taken, err := workspaces.Scoped[models.Category](ctx, tx).Where("name", name).Exists(ctx)
		if err != nil {
			return err
		}

		if taken {
			return errName
		}

		c := models.Category{Name: name, Kind: kind, ParentID: parent, Color: in.Color, Icon: in.Icon}
		if err := orm.Create(ctx, tx, &c); err != nil {
			// Two devices that create the same name at once: the partial unique
			// index on live names settles it, and the loser gets the same answer.
			if s.DB.UniqueViolation(err) {
				return errName
			}

			return err
		}

		id = c.ID

		return nil
	})
	if err != nil {
		return models.Category{}, err
	}

	return reload[models.Category](ctx, s.DB, id)
}

// UpdateCategory renames a category and changes its parent, color and icon.
// The kind never changes. ParentID 0 makes it top-level.
func (s *Service) UpdateCategory(ctx context.Context, id int64, in CategoryInput) (models.Category, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return models.Category{}, errName
	}

	if err := checkLook(in.Color, in.Icon); err != nil {
		return models.Category{}, err
	}

	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Category{}, err
	}

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		cur, err := workspaces.Scoped[models.Category](ctx, tx).Where("id", id).First(ctx)
		if err != nil {
			return err
		}

		kids, err := workspaces.Scoped[models.Category](ctx, tx).Where("parent_id", id).Exists(ctx)
		if err != nil {
			return err
		}

		parent := cur.ParentID

		// Keeping the parent it already has is not a change, even if that parent was archived since.
		if !cur.ParentID.Valid || cur.ParentID.V != in.ParentID {
			if parent, err = parentFor(ctx, tx, in.ParentID, id, cur.Kind, kids); err != nil {
				return err
			}
		}

		taken, err := workspaces.Scoped[models.Category](ctx, tx).Where("name", name).Where("id", "!=", id).Exists(ctx)
		if err != nil {
			return err
		}

		if taken {
			return errName
		}

		var p any
		if parent.Valid {
			p = parent.V
		}

		_, err = workspaces.Scoped[models.Category](ctx, tx).Where("id", id).Update(ctx, orm.Set{
			"name": name, "parent_id": p, "color": in.Color, "icon": in.Icon,
		})
		if s.DB.UniqueViolation(err) {
			return errName
		}

		return err
	})
	if err != nil {
		return models.Category{}, err
	}

	return reload[models.Category](ctx, s.DB, id)
}

// ArchiveCategory hides a category from the pickers; its transactions and
// budget stay. Its subcategories are archived with it. Archiving twice
// changes nothing.
func (s *Service) ArchiveCategory(ctx context.Context, id int64) (models.Category, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Category{}, err
	}

	at := s.now().UTC()

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		c, err := reload[models.Category](ctx, tx, id)
		if err != nil || c.ArchivedAt.Valid {
			return err
		}

		_, err = workspaces.Scoped[models.Category](ctx, tx).WhereNull("archived_at").
			WhereGroup(func(g orm.Query[models.Category]) orm.Query[models.Category] {
				return g.Where("id", id).OrWhere("parent_id", id)
			}).Update(ctx, orm.Set{"archived_at": at})

		return err
	})
	if err != nil {
		return models.Category{}, err
	}

	return reload[models.Category](ctx, s.DB, id)
}

// UnarchiveCategory brings a category back, with the subcategories that were
// archived together with it. A subcategory needs its parent unarchived first.
func (s *Service) UnarchiveCategory(ctx context.Context, id int64) (models.Category, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Category{}, err
	}

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		c, err := reload[models.Category](ctx, tx, id)
		if err != nil || !c.ArchivedAt.Valid {
			return err
		}

		if c.ParentID.Valid {
			archived, err := workspaces.Scoped[models.Category](ctx, tx).Where("id", c.ParentID.V).
				WhereNotNull("archived_at").Exists(ctx)
			if err != nil {
				return err
			}

			if archived {
				return errParentArchived
			}
		}

		if _, err := workspaces.Scoped[models.Category](ctx, tx).Where("parent_id", id).
			Where("archived_at", c.ArchivedAt.V).Update(ctx, orm.Set{"archived_at": nil}); err != nil {
			return err
		}

		_, err = workspaces.Scoped[models.Category](ctx, tx).Where("id", id).Update(ctx, orm.Set{"archived_at": nil})

		return err
	})
	if err != nil {
		return models.Category{}, err
	}

	return reload[models.Category](ctx, s.DB, id)
}

// DeleteCategoryReassigning deletes a category after moving what used it to
// another category of the same kind, in one database transaction: its
// transactions (never a transfer leg, which has no category) and its budget.
// If the target already has a budget the target keeps its own and the budget
// of the deleted category is deleted. A category that nothing uses may be
// deleted without a target; one that something uses is refused without one.
// A category with subcategories is refused.
func (s *Service) DeleteCategoryReassigning(ctx context.Context, id, reassignTo int64) error {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return err
	}

	at := s.now().UTC()

	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		c, err := reload[models.Category](ctx, tx, id)
		if err != nil {
			return err
		}

		kids, err := workspaces.Scoped[models.Category](ctx, tx).Where("parent_id", id).Exists(ctx)
		if err != nil {
			return err
		}

		if kids {
			return errCategoryChildren
		}

		used, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("category_id", id).Exists(ctx)
		if err != nil {
			return err
		}

		budget, err := workspaces.Scoped[models.Budget](ctx, tx).Where("category_id", id).First(ctx)
		if err != nil && !notFound(err) {
			return err
		}

		hasBudget := err == nil

		if reassignTo == 0 && (used || hasBudget) {
			return errReassign
		}

		if reassignTo > 0 {
			if err := moveCategoryUse(ctx, tx, c, reassignTo, budget, hasBudget, at); err != nil {
				return err
			}
		} else if hasBudget {
			if _, err := workspaces.Scoped[models.Budget](ctx, tx).Where("id", budget.ID).
				Update(ctx, orm.Set{"deleted_at": at}); err != nil {
				return err
			}
		}

		return softDelete[models.Category](ctx, tx, id, at)
	})
}

func moveCategoryUse(
	ctx context.Context, tx *database.Tx, from models.Category, to int64, budget models.Budget, hasBudget bool, at time.Time,
) error {
	target, err := workspaces.Scoped[models.Category](ctx, tx).Where("id", to).First(ctx)
	if err != nil {
		return orElse(err, errReassign)
	}

	if target.ID == from.ID || target.Kind != from.Kind || target.ArchivedAt.Valid {
		return errReassign
	}

	if _, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("category_id", from.ID).
		WhereNull("transfer_uid").Update(ctx, orm.Set{"category_id": to}); err != nil {
		return err
	}

	if !hasBudget {
		return nil
	}

	taken, err := workspaces.Scoped[models.Budget](ctx, tx).Where("category_id", to).Exists(ctx)
	if err != nil {
		return err
	}

	set := orm.Set{"category_id": to}
	if taken {
		set = orm.Set{"deleted_at": at}
	}

	_, err = workspaces.Scoped[models.Budget](ctx, tx).Where("id", budget.ID).Update(ctx, set)

	return err
}

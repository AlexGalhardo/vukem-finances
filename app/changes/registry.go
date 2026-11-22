package changes

import (
	"fmt"
	"slices"
	"sync"
)

// Ref names a parent an entity points at: Column holds the parent's integer
// id in the database and travels as "<entity>_uid" on the wire.
type Ref struct {
	Column string
	Entity string
}

// Entity is a synced entity: its table, the columns a client may write, and
// its parents.
type Entity struct {
	Name   string
	Table  string
	Fields []string
	Refs   []Ref
	// JSON lists the Fields whose column holds a JSON document (the tag_uids
	// array of a transaction); a snapshot sends it decoded, and a change to it
	// is one field for Merge, so two edits of the same list are a conflict.
	JSON []string
}

// tracked lists the columns whose change is logged: the writable ones and the
// tombstone.
func (e Entity) tracked() []string { return append(slices.Clone(e.Fields), "deleted_at") }

var (
	mu       sync.RWMutex
	entities []Entity //nolint:gochecknoglobals // the registry of synced entities, filled at init
)

// Register adds a synced entity; later epics register their tables here. It
// panics on a duplicate name or table, because that is a programming error
// found at start-up.
func Register(e Entity) {
	mu.Lock()
	defer mu.Unlock()

	for _, o := range entities {
		if o.Name == e.Name || o.Table == e.Table {
			panic(fmt.Sprintf("changes: entity %s (table %s) registered twice", e.Name, e.Table))
		}
	}

	entities = append(entities, e)
}

// Entities returns the registered entities in registration order.
func Entities() []Entity {
	mu.RLock()
	defer mu.RUnlock()

	return slices.Clone(entities)
}

// Lookup finds an entity by its wire name.
func Lookup(name string) (Entity, bool) {
	for _, e := range Entities() {
		if e.Name == name {
			return e, true
		}
	}

	return Entity{}, false
}

func init() {
	Register(Entity{
		Name: "account", Table: "accounts",
		Fields: []string{"name", "kind", "opening_cents", "archived_at", "position"},
	})
	Register(Entity{
		Name: "category", Table: "categories",
		Fields: []string{"name", "kind", "parent_id", "color", "icon", "archived_at"},
		Refs:   []Ref{{"parent_id", "category"}},
	})
	Register(Entity{Name: "tag", Table: "tags", Fields: []string{"name", "color"}})
	Register(Entity{
		Name: "transaction", Table: "transactions",
		Fields: []string{
			"account_id", "category_id", "kind", "amount_cents", "description", "occurred_on", "transfer_uid", "tag_uids",
			"statement_ref", "recurrence_uid", "occurrence_on", "installment_group_uid", "installment_no", "installment_total",
		},
		Refs: []Ref{{"account_id", "account"}, {"category_id", "category"}},
		JSON: []string{"tag_uids"},
	})
	Register(Entity{
		Name: "budget", Table: "budgets",
		Fields: []string{"category_id", "limit_cents", "rollover"},
		Refs:   []Ref{{"category_id", "category"}},
	})
	Register(Entity{
		Name: "credit_card", Table: "credit_cards",
		Fields: []string{"account_id", "limit_cents", "closing_day", "due_day", "payment_account_id", "brand", "last4"},
		Refs:   []Ref{{"account_id", "account"}, {"payment_account_id", "account"}},
	})
	Register(Entity{
		Name: "recurrence", Table: "recurrences",
		Fields: []string{
			"account_id", "category_id", "kind", "amount_cents", "description", "frequency", "interval",
			"start_date", "end_date", "max_count", "generated_through", "active",
		},
		Refs: []Ref{{"account_id", "account"}, {"category_id", "category"}},
	})
	Register(Entity{
		Name: "goal", Table: "goals",
		Fields: []string{"name", "target_cents", "target_date", "archived_at"},
	})
	Register(Entity{
		Name: "goal_contribution", Table: "goal_contributions",
		Fields: []string{"goal_id", "amount_cents", "on_date", "note"},
		Refs:   []Ref{{"goal_id", "goal"}},
	})
}

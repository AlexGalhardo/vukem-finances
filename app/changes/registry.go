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
	Register(Entity{Name: "category", Table: "categories", Fields: []string{"name", "kind"}})
	Register(Entity{
		Name: "transaction", Table: "transactions",
		Fields: []string{"account_id", "category_id", "kind", "amount_cents", "description", "occurred_on", "transfer_uid"},
		Refs:   []Ref{{"account_id", "account"}, {"category_id", "category"}},
	})
	Register(Entity{
		Name: "budget", Table: "budgets",
		Fields: []string{"category_id", "limit_cents"},
		Refs:   []Ref{{"category_id", "category"}},
	})
}

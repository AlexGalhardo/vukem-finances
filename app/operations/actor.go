package operations

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Surface is a way into the registry.
type Surface string

// The surfaces. Epics 31 and 32 add the CLI and MCP adapters.
const (
	SurfaceREST Surface = "rest"
	SurfaceCLI  Surface = "cli"
	SurfaceMCP  Surface = "mcp"
)

// Surfaces lists every surface, in the order the inventory shows them.
func Surfaces() []Surface { return []Surface{SurfaceREST, SurfaceCLI, SurfaceMCP} }

// Actor is who an operation acts as: the person, the workspace, the role the
// person holds there now and the scopes the credential carries. Its fields are
// private: only an authenticated adapter (NewActor, after it checked the
// credential and the membership) or the local owner resolver of the personal
// edition (LocalOwner) can make one, and an operation reads the workspace from
// here, never from its input.
type Actor struct {
	surface    Surface
	user       models.User
	workspace  int64
	role       string
	scopes     []string
	credential string
}

// NewActor builds the actor of an authenticated call. credential names what
// authenticated it (the id of a token) and keys its idempotency keys; the
// caller must already have checked it, and the membership behind role.
func NewActor(surface Surface, user models.User, workspaceID int64, role string, scopes []string, credential string) Actor {
	return Actor{
		surface: surface, user: user, workspace: workspaceID, role: role,
		scopes: slices.Clone(scopes), credential: credential,
	}
}

// User is the person who acts.
func (a Actor) User() models.User { return a.user }

// WorkspaceID is the workspace the actor works in.
func (a Actor) WorkspaceID() int64 { return a.workspace }

// Role is the role of the person in the workspace.
func (a Actor) Role() string { return a.role }

// Scopes are the scopes of the credential.
func (a Actor) Scopes() []string { return slices.Clone(a.scopes) }

// Surface is the adapter that built the actor.
func (a Actor) Surface() Surface { return a.surface }

// HasScope reports whether the credential carries scope.
func (a Actor) HasScope(scope string) bool { return slices.Contains(a.scopes, scope) }

func (a Actor) valid() bool {
	return a.surface != "" && a.user.ID > 0 && a.workspace > 0 && a.role != ""
}

// idempotencyScope separates callers, so one cannot replay another's key.
func (a Actor) idempotencyScope() string {
	if a.credential != "" {
		return string(a.surface) + "-credential:" + a.credential
	}

	return string(a.surface) + "-user:" + strconv.FormatInt(a.user.ID, 10) + ":" + strconv.FormatInt(a.workspace, 10)
}

// LocalOwner is the actor of the person at the console of a personal
// installation: its owner, in its one workspace, with the scopes given. It is
// how the CLI acts without a token. Only the personal edition has it: a saas
// server has no local owner, and a surface there needs a token.
func LocalOwner(ctx context.Context, ws *workspaces.Service, surface Surface, scopes []string) (Actor, error) {
	if !ws.Edition.Personal() {
		return Actor{}, fail(http.StatusForbidden, CodeForbidden, "There is no local owner outside the personal edition: use a token.")
	}

	m, err := orm.QueryOf[models.Membership](ws.DB).Where("role", models.RoleOwner).OrderBy("id", orm.Asc).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return Actor{}, fail(http.StatusNotFound, CodeNotFound, "This installation has no owner yet: finish the setup first.")
	}

	if err != nil {
		return Actor{}, err
	}

	u, err := orm.QueryOf[models.User](ws.DB).Find(ctx, m.UserID)
	if err != nil {
		return Actor{}, err
	}

	return NewActor(surface, u, m.WorkspaceID, m.Role, scopes, ""), nil
}

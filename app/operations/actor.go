package operations

import (
	"slices"
	"strconv"

	"github.com/AlexGalhardo/vukem-finances/app/models"
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

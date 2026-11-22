package oauth

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/oauth"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Consent is the sign-in and the consent screen of the authorization endpoint.
// The framework owns every security decision (client, redirect, PKCE, resource,
// the signed form); this owns who is signed in, which workspace they are
// granting, and the screen.
//
// The route is behind the web login, the verified-email check and the tenant
// middleware, so the person and the current workspace are on the request. The
// workspace is part of the subject the CSRF token of the form is bound to:
// picking another one goes through PickWorkspace and comes back here with a new
// form.
type Consent struct {
	WS *workspaces.Service
	// Scopes are the scopes the server offers, for the labels of the screen.
	Scopes []string
}

// Subject implements oauth.Consent: the signed-in person and the workspace
// they are in now, as long as they are a member of it.
func (c *Consent) Subject(w http.ResponseWriter, r *http.Request) (string, bool) {
	u, err := auth.UserFrom[models.User](r.Context(), "web")
	if err != nil {
		http.Error(w, "sign in to continue", http.StatusUnauthorized)

		return "", false
	}

	ws, ok := workspaces.TenantID(r.Context())
	if !ok {
		http.Error(w, "no workspace", http.StatusForbidden)

		return "", false
	}

	role, member, err := c.WS.MemberRole(r.Context(), u.ID, ws)
	if err != nil || !member {
		http.Error(w, "you are not a member of this workspace", http.StatusForbidden)

		return "", false
	}

	// The screen shows what the role cannot grant; the decision is enforced here.
	// The scopes of the POST are the ones the signed form carried, and the
	// framework grants exactly those, so one the role lacks refuses the decision.
	if r.Method == http.MethodPost {
		for _, sc := range strings.Fields(r.PostFormValue("scope")) {
			if !workspaces.RoleCan(role, operations.AbilityOf(sc)) {
				http.Error(w, "your role cannot grant "+sc, http.StatusForbidden)

				return "", false
			}
		}
	}

	return Subject{UserID: u.ID, WorkspaceID: ws}.String(), true
}

// Render implements oauth.Consent.
func (c *Consent) Render(w http.ResponseWriter, r *http.Request, req oauth.ConsentRequest) error {
	sub, ok := ParseSubject(req.Subject)
	if !ok {
		http.Error(w, "invalid subject", http.StatusBadRequest)

		return nil
	}

	ctx := r.Context()

	u, err := auth.UserFrom[models.User](ctx, "web")
	if err != nil {
		return err
	}

	role, _, err := c.WS.MemberRole(ctx, sub.UserID, sub.WorkspaceID)
	if err != nil {
		return err
	}

	mine, err := c.WS.Mine(ctx, u)
	if err != nil {
		return err
	}

	props := pages.OAuthConsentProps{
		ClientName: req.Client.Name, ClientID: req.Client.ID, RedirectHost: redirectHost(req.Fields.Get("redirect_uri")),
		Role: role, Action: req.Action, ReturnTo: r.URL.RequestURI(),
	}

	props.ClientName, props.ClaimedName = DisplayNames(req.Client)

	for _, ws := range mine {
		current := ws.ID == sub.WorkspaceID
		if current {
			props.Workspace = ws.Name
		}

		props.Workspaces = append(props.Workspaces, pages.ConsentWorkspace{ID: strconv.FormatInt(ws.ID, 10), Name: ws.Name, Current: current})
	}

	for _, s := range req.Scopes {
		props.Scopes = append(props.Scopes, pages.ConsentScope{
			Name:    s,
			Allowed: workspaces.RoleCan(role, operations.AbilityOf(s)),
			Writes:  strings.HasSuffix(s, ":write"),
		})
	}

	names := make([]string, 0, len(req.Fields))
	for name := range req.Fields {
		names = append(names, name)
	}

	slices.Sort(names)

	for _, name := range names {
		for _, v := range req.Fields[name] {
			props.Fields = append(props.Fields, pages.ConsentField{Name: name, Value: v})
		}
	}

	w.Header().Set("Cache-Control", "no-store")

	return pages.OAuthConsent.Render(web.NewCtx(w, r, web.ServicesFrom(ctx)), props)
}

// DisplayNames are what the screen calls a client. A client identified by the
// address of a metadata document can claim any name, so its verified host is the
// name and what it claims is returned apart, to be shown as unverified. A client
// registered with an id that is not an address keeps the name it was registered
// with.
func DisplayNames(c *oauth.Client) (name, claimed string) {
	host := ClientName(c.ID)

	if host == c.ID {
		if c.Name != "" {
			return c.Name, ""
		}

		return c.ID, ""
	}

	if c.Name != "" && c.Name != host {
		return host, c.Name
	}

	return host, ""
}

// redirectHost is the host the browser returns to, shown so the person can see
// where the answer goes.
func redirectHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}

	return u.Host
}

// PickWorkspace makes the workspace of the form the current one and returns to
// the authorization request. The return address must be the authorization
// endpoint of this server and nothing else, so the form is no open redirect.
func PickWorkspace(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PostFormValue("workspace"), 10, 64)
	back := r.PostFormValue("return")

	if err != nil || !strings.HasPrefix(back, "/authorize?") || strings.ContainsAny(back, "\r\n") || strings.HasPrefix(back, "//") {
		http.Error(w, "invalid request", http.StatusBadRequest)

		return
	}

	if err := workspaces.Switch(r, id); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)

		return
	}

	http.Redirect(w, r, back, http.StatusSeeOther)
}

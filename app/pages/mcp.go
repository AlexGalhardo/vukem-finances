package pages

import "github.com/AlexGalhardo/vukem/inertia"

// OAuthConsent asks the signed-in person whether an AI client may act in one
// workspace of theirs (epic 33). The form posts to Action with every Field as a
// hidden input and one more, decision, set by the button.
var OAuthConsent = inertia.Define[OAuthConsentProps]("OAuthConsent")

// ConnectedApps lists the AI clients the person connected to the current
// workspace, with what they may do, and revokes them.
var ConnectedApps = inertia.Define[ConnectedAppsProps]("ConnectedApps")

// ConsentScope is one scope the client asks for. Allowed is false when the role
// of the person in the workspace does not let them grant it: the scope is then
// shown as not available and is never honored.
type ConsentScope struct {
	Name    string `json:"name"`
	Allowed bool   `json:"allowed"`
	// Writes is true for a scope that changes data.
	Writes bool `json:"writes"`
}

// ConsentWorkspace is a workspace the person can pick.
type ConsentWorkspace struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Current bool   `json:"current"`
}

// ConsentField is a hidden input of the consent form.
type ConsentField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// OAuthConsentProps are the props of OAuthConsent.
type OAuthConsentProps struct {
	// ClientName is what the client calls itself; ClientID is its identifier,
	// the verified identity: the address of its metadata document or the id it
	// was registered with. RedirectHost is where the browser returns to.
	ClientName string `json:"clientName"`
	// ClaimedName is the name a client identified by a metadata document gives
	// itself. Nothing verified it: the screen shows it labeled as unverified.
	ClaimedName  string             `json:"claimedName"`
	ClientID     string             `json:"clientId"`
	RedirectHost string             `json:"redirectHost"`
	Workspace    string             `json:"workspace"`
	Role         string             `json:"role"`
	Workspaces   []ConsentWorkspace `json:"workspaces"`
	Scopes       []ConsentScope     `json:"scopes"`
	Action       string             `json:"action"`
	Fields       []ConsentField     `json:"fields"`
	// ReturnTo is this page's address, where picking another workspace comes back to.
	ReturnTo string `json:"returnTo"`
}

// ConnectedApp is a grant as the page shows it. Status is live or revoked.
type ConnectedApp struct {
	ID         string   `json:"id"`
	Client     string   `json:"client"`
	Scopes     []string `json:"scopes"`
	CreatedAt  string   `json:"createdAt"`
	LastUsedAt string   `json:"lastUsedAt"`
	Status     string   `json:"status"`
}

// ConnectedAppsProps are the props of ConnectedApps.
type ConnectedAppsProps struct {
	Apps []ConnectedApp `json:"apps"`
}

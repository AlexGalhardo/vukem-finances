package pages

import (
	"github.com/AlexGalhardo/vukem/inertia"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// The typed pages of the finance app. Dashboard is declared in auth.go.
var (
	Transactions   = inertia.Define[TransactionsProps]("Transactions")
	Accounts       = inertia.Define[AccountsProps]("Accounts")
	Categories     = inertia.Define[CategoriesProps]("Categories")
	Budgets        = inertia.Define[BudgetsProps]("Budgets")
	Import         = inertia.Define[ImportProps]("Import")
	Workspace      = inertia.Define[WorkspaceProps]("Workspace")
	InviteAccept   = inertia.Define[InviteAcceptProps]("InviteAccept")
	InviteRegister = inertia.Define[InviteRegisterProps]("InviteRegister")
	Billing        = inertia.Define[BillingProps]("Billing")
	AdminRevenue   = inertia.Define[AdminRevenueProps]("Admin/Revenue")
)

// WorkspaceView is a workspace as the switcher shows it.
type WorkspaceView struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Timezone string `json:"timezone"`
}

// SharedWorkspace is the workspace data every page of a signed-in user
// receives: the current workspace, the others for the switcher, the role
// of the user and the plan.
type SharedWorkspace struct {
	Current WorkspaceView   `json:"current"`
	All     []WorkspaceView `json:"all"`
	Role    string          `json:"role"`
	Plan    string          `json:"plan"`
	Admin   bool            `json:"admin"`
	Edition string          `json:"edition"` // personal or saas
}

// DashboardProps are the props of Dashboard.
type DashboardProps struct {
	Dashboard finance.Dashboard `json:"dashboard"`
}

// TransactionsProps are the props of Transactions.
type TransactionsProps struct {
	Transactions []finance.TxRow   `json:"transactions"`
	Accounts     []models.Account  `json:"accounts"`
	Categories   []models.Category `json:"categories"`
	Filters      TxFilters         `json:"filters"`
	Today        string            `json:"today"`
}

// TxFilters echoes the filters of the list.
type TxFilters struct {
	From       string `json:"from"`
	To         string `json:"to"`
	CategoryID string `json:"categoryId"`
	AccountID  string `json:"accountId"`
	Kind       string `json:"kind"`
}

// AccountsProps are the props of Accounts.
type AccountsProps struct {
	Accounts []models.Account  `json:"accounts"`
	Limits   workspaces.Limits `json:"limits"`
}

// CategoriesProps are the props of Categories.
type CategoriesProps struct {
	Categories []models.Category `json:"categories"`
}

// BudgetsProps are the props of Budgets.
type BudgetsProps struct {
	Budgets    []finance.BudgetStatus `json:"budgets"`
	Categories []models.Category      `json:"categories"`
	Month      string                 `json:"month"`
	Enabled    bool                   `json:"enabled"`
}

// ImportProps are the props of Import.
type ImportProps struct {
	Accounts    []models.Account `json:"accounts"`
	Imports     []models.Import  `json:"imports"`
	Preview     *finance.Preview `json:"preview"`
	DateFormats []string         `json:"dateFormats"`
	Enabled     bool             `json:"enabled"`
}

// WorkspaceProps are the props of Workspace.
type WorkspaceProps struct {
	WorkspaceDetails WorkspaceView       `json:"workspaceDetails"`
	Members          []workspaces.Member `json:"members"`
	Invites          []models.Invite     `json:"invites"`
	Limits           workspaces.Limits   `json:"limits"`
	CanManage        bool                `json:"canManage"`
	// Audit is the latest sensitive actions; empty unless the user manages members.
	Audit []workspaces.AuditEntry `json:"audit"`
}

// InviteAcceptProps are the props of InviteAccept.
type InviteAcceptProps struct {
	Token string `json:"token"`
}

// InviteRegisterProps are the props of InviteRegister: the address is the
// one the invite was sent to and cannot be changed.
type InviteRegisterProps struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

// BillingProps are the props of Billing.
type BillingProps struct {
	Limits       workspaces.Limits `json:"limits"`
	Subscription *BillingStatus    `json:"subscription"`
	TrialDays    int               `json:"trialDays"`
	CanManage    bool              `json:"canManage"`
	Configured   bool              `json:"configured"`
	Currencies   []string          `json:"currencies"`
	Checkout     bool              `json:"checkout"`
}

// BillingStatus is the Stripe subscription of the workspace.
type BillingStatus struct {
	Status            string `json:"status"`
	Valid             bool   `json:"valid"`
	CurrentPeriodEnd  string `json:"currentPeriodEnd"`
	CancelAtPeriodEnd bool   `json:"cancelAtPeriodEnd"`
}

// AdminRevenueProps are the props of Admin/Revenue.
type AdminRevenueProps struct {
	MRRCents   int64 `json:"mrrCents"`
	Active     int64 `json:"active"`
	Trialing   int64 `json:"trialing"`
	PastDue    int64 `json:"pastDue"`
	Workspaces int64 `json:"workspaces"`
	Users      int64 `json:"users"`
}

// AI is the settings page of the AI layer.
var AI = inertia.Define[AIProps]("AI")

// AICredentialView is the saved key as the page may show it: that it exists,
// who and when, and the last four characters. The key itself and its
// ciphertext are not in it.
type AICredentialView struct {
	ID         int64  `json:"id"`
	Provider   string `json:"provider"`
	Label      string `json:"label"`
	KeyHint    string `json:"keyHint"`
	BaseURL    string `json:"baseUrl"`
	Model      string `json:"model"`
	CreatedAt  string `json:"createdAt"`
	LastUsedAt string `json:"lastUsedAt"`
}

// AIProps are the props of AI. Credential is nil until a key is saved; only
// the owner receives it.
type AIProps struct {
	Settings   ai.Settings       `json:"settings"`
	Credential *AICredentialView `json:"credential"`
	Usage      ai.MonthUsage     `json:"usage"`
	Providers  []string          `json:"providers"`
	CanManage  bool              `json:"canManage"`
	// PrivateHosts says whether this server lets the compatible provider reach a local network.
	PrivateHosts bool `json:"privateHosts"`
}

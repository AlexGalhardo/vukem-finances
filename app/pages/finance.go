package pages

import (
	"github.com/AlexGalhardo/vukem/inertia"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/alerts"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// The typed pages of the finance app. Dashboard is declared in auth.go.
var (
	Transactions   = inertia.Define[TransactionsProps]("Transactions")
	Accounts       = inertia.Define[AccountsProps]("Accounts")
	Cards          = inertia.Define[CardsProps]("Cards")
	CardStatement  = inertia.Define[CardStatementProps]("CardStatement")
	Categories     = inertia.Define[CategoriesProps]("Categories")
	Budgets        = inertia.Define[BudgetsProps]("Budgets")
	Goals          = inertia.Define[GoalsProps]("Goals")
	Notifications  = inertia.Define[NotificationsProps]("Notifications")
	Recurrences    = inertia.Define[RecurrencesProps]("Recurrences")
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
	Dashboard finance.Dashboard      `json:"dashboard"`
	Upcoming  []finance.UpcomingItem `json:"upcoming"`
}

// RecurrencesProps are the props of Recurrences: the rules and what is due
// in the next 30 days.
type RecurrencesProps struct {
	Recurrences []models.Recurrence    `json:"recurrences"`
	Upcoming    []finance.UpcomingItem `json:"upcoming"`
	Accounts    []models.Account       `json:"accounts"`
	Categories  []models.Category      `json:"categories"`
	Today       string                 `json:"today"`
}

// TransactionsProps are the props of Transactions.
type TransactionsProps struct {
	Transactions []finance.TxRow   `json:"transactions"`
	Accounts     []models.Account  `json:"accounts"`
	Categories   []models.Category `json:"categories"`
	Tags         []models.Tag      `json:"tags"`
	Filters      TxFilters         `json:"filters"`
	Today        string            `json:"today"`
	// Total is the size of the whole filtered result; Page and PageSize say which slice Transactions is.
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

// TxFilters echoes the filters of the list.
type TxFilters struct {
	From       string `json:"from"`
	To         string `json:"to"`
	CategoryID string `json:"categoryId"`
	AccountID  string `json:"accountId"`
	Kind       string `json:"kind"`
	Q          string `json:"q"`
	TagID      string `json:"tagId"`
	Min        string `json:"min"`
	Max        string `json:"max"`
}

// AccountsProps are the props of Accounts.
type AccountsProps struct {
	Accounts []models.Account  `json:"accounts"`
	Limits   workspaces.Limits `json:"limits"`
	// Balances has one entry per account, as of today; NetWorthCents is the
	// sum of the ones that are not archived.
	Balances      []finance.AccountBalance `json:"balances"`
	NetWorthCents int64                    `json:"netWorthCents"`
	Kinds         []string                 `json:"kinds"`
}

// CardsProps are the props of Cards.
type CardsProps struct {
	Cards []finance.CardOverview `json:"cards"`
	// Accounts are the accounts a statement can be paid from and a card can
	// name as its default (no credit cards, no archived accounts).
	Accounts []models.Account  `json:"accounts"`
	Brands   []string          `json:"brands"`
	Limits   workspaces.Limits `json:"limits"`
}

// CardStatementProps are the props of CardStatement: one card, its
// statements newest first and the one selected with its rows.
type CardStatementProps struct {
	Card       finance.CardOverview       `json:"card"`
	Statements []finance.StatementSummary `json:"statements"`
	Statement  finance.StatementDetail    `json:"statement"`
	Accounts   []models.Account           `json:"accounts"`
	Brands     []string                   `json:"brands"`
	Today      string                     `json:"today"`
}

// CategoriesProps are the props of Categories.
type CategoriesProps struct {
	Categories []models.Category `json:"categories"`
	Tags       []models.Tag      `json:"tags"`
	// Colors and Icons are the lists a category may pick from; MaxTags is the tag limit of a workspace.
	Colors  []string `json:"colors"`
	Icons   []string `json:"icons"`
	MaxTags int      `json:"maxTags"`
}

// BudgetsProps are the props of Budgets.
type BudgetsProps struct {
	Budgets    []finance.BudgetStatus `json:"budgets"`
	Categories []models.Category      `json:"categories"`
	Month      string                 `json:"month"`
	Enabled    bool                   `json:"enabled"`
}

// GoalsProps are the props of Goals: the goals with their progress and the
// newest contributions of each (the client groups them by goal_id).
type GoalsProps struct {
	Goals         []finance.GoalStatus      `json:"goals"`
	Contributions []models.GoalContribution `json:"contributions"`
	Today         string                    `json:"today"`
}

// NotificationsProps are the props of Notifications: the full list of the
// signed-in person and their channel choices.
type NotificationsProps struct {
	Items       []alerts.View          `json:"items"`
	Preferences []alerts.PreferenceRow `json:"preferences"`
}

// ImportProps are the props of Import.
type ImportProps struct {
	Accounts    []models.Account     `json:"accounts"`
	Imports     []finance.ImportView `json:"imports"`
	Preview     *finance.Preview     `json:"preview"`
	DateFormats []string             `json:"dateFormats"`
	Enabled     bool                 `json:"enabled"`
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
	// Pix is nil while PIX is not configured.
	Pix *PixOffer `json:"pix"`
	// PixAccess is the PIX period of the workspace; nil when it never paid by PIX.
	PixAccess *PixAccess `json:"pixAccess"`
}

// BillingStatus is the Stripe subscription of the workspace. RevokedReason is
// refund, dispute, dispute_lost or admin when access was taken away.
type BillingStatus struct {
	Status            string `json:"status"`
	Valid             bool   `json:"valid"`
	CurrentPeriodEnd  string `json:"currentPeriodEnd"`
	CancelAtPeriodEnd bool   `json:"cancelAtPeriodEnd"`
	RevokedReason     string `json:"revokedReason"`
}

// PixOffer is what the PIX dialog may sell: the months on offer and their
// total in cents, computed by the server.
type PixOffer struct {
	Currency string      `json:"currency"`
	Options  []PixOption `json:"options"`
}

// PixOption is one period and its price.
type PixOption struct {
	Months     int   `json:"months"`
	TotalCents int64 `json:"totalCents"`
}

// PixAccess is the PIX period of a workspace. It never renews by itself.
type PixAccess struct {
	Active        bool   `json:"active"`
	ValidUntil    string `json:"validUntil"`
	RevokedReason string `json:"revokedReason"`
}

// PixView is a PIX charge as the dialog shows and polls it.
type PixView struct {
	Ref         string `json:"ref"`
	Status      string `json:"status"`
	Months      int    `json:"months"`
	AmountCents int64  `json:"amountCents"`
	BRCode      string `json:"brCode,omitempty"`
	QR          string `json:"qr,omitempty"`
	ExpiresAt   string `json:"expiresAt"`
	Plan        string `json:"plan,omitempty"`
}

// AdminRevenueProps are the props of Admin/Revenue.
type AdminRevenueProps struct {
	MRRCents int64 `json:"mrrCents"`
	// PixMRRCents is the monthly equivalent of the valid PIX payments, in BRL cents.
	PixMRRCents int64             `json:"pixMrrCents"`
	PixActive   int64             `json:"pixActive"`
	Active      int64             `json:"active"`
	Trialing    int64             `json:"trialing"`
	PastDue     int64             `json:"pastDue"`
	Workspaces  int64             `json:"workspaces"`
	Users       int64             `json:"users"`
	RecentPix   []AdminPixPayment `json:"recentPix"`
}

// AdminPixPayment is a PIX payment in the admin overview. Workspace is its id,
// not a name or a person.
type AdminPixPayment struct {
	Ref         string `json:"ref"`
	Workspace   int64  `json:"workspace"`
	Status      string `json:"status"`
	AmountCents int64  `json:"amountCents"`
	Months      int    `json:"months"`
	CanRefund   bool   `json:"canRefund"`
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

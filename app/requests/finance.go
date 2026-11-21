package requests

import "github.com/AlexGalhardo/vukem/z"

// The finance forms. Ids and amounts arrive as text (select values, "12.50")
// and are parsed by the finance service, which owns the money rules.

// Transaction is the input of the new transaction form.
type Transaction struct {
	AccountID   string `json:"accountId"   form:"accountId"`
	CategoryID  string `json:"categoryId"  form:"categoryId"`
	Kind        string `json:"kind"        form:"kind"`
	Amount      string `json:"amount"      form:"amount"`
	Description string `json:"description" form:"description"`
	Date        string `json:"date"        form:"date"`
	// Tags are tag names, created when new. Absent leaves the tags of an
	// edited transaction alone; a list, even an empty one, replaces them.
	Tags []string `json:"tags" form:"tags"`
}

// TransactionSchema validates Transaction.
var TransactionSchema = z.Struct[Transaction](z.Shape{
	"accountId":   z.String().Trim().Min(1).Max(20),
	"categoryId":  z.String().Trim().Max(20).Optional(),
	"kind":        z.String().Trim().Min(1).Max(20),
	"amount":      z.String().Trim().Min(1).Max(30),
	"description": z.String().Trim().Max(200).Optional(),
	"date":        z.String().Trim().Min(10).Max(10),
	"tags":        z.Slice(z.String().Trim().Max(120)).Max(40).Optional(),
})

// Account is the input of the new account form.
type Account struct {
	Name    string `json:"name"    form:"name"`
	Kind    string `json:"kind"    form:"kind"`
	Opening string `json:"opening" form:"opening"`
}

// AccountSchema validates Account.
var AccountSchema = z.Struct[Account](z.Shape{
	"name":    z.String().Trim().Min(1).Max(80),
	"kind":    z.String().Trim().Min(1).Max(20),
	"opening": z.String().Trim().Max(30).Optional(),
})

// Category is the input of the new category form.
type Category struct {
	Name string `json:"name" form:"name"`
	// Kind is needed for a top-level category and ignored on an edit: a
	// category never changes kind, and a subcategory takes its parent's.
	Kind     string `json:"kind"     form:"kind"`
	ParentID string `json:"parentId" form:"parentId"`
	Color    string `json:"color"    form:"color"`
	Icon     string `json:"icon"     form:"icon"`
}

// CategorySchema validates Category.
var CategorySchema = z.Struct[Category](z.Shape{
	"name":     z.String().Trim().Min(1).Max(60),
	"kind":     z.String().Trim().Max(20).Optional(),
	"parentId": z.String().Trim().Max(20).Optional(),
	"color":    z.String().Trim().Max(20).Optional(),
	"icon":     z.String().Trim().Max(20).Optional(),
})

// CategoryDelete is the input of the delete category form.
type CategoryDelete struct {
	ReassignTo string `json:"reassignTo" form:"reassignTo"`
}

// CategoryDeleteSchema validates CategoryDelete.
var CategoryDeleteSchema = z.Struct[CategoryDelete](z.Shape{
	"reassignTo": z.String().Trim().Max(20).Optional(),
})

// Tag is the input of the tag form.
type Tag struct {
	Name  string `json:"name"  form:"name"`
	Color string `json:"color" form:"color"`
}

// TagSchema validates Tag.
var TagSchema = z.Struct[Tag](z.Shape{
	"name":  z.String().Trim().Min(1).Max(120),
	"color": z.String().Trim().Max(20).Optional(),
})

// Budget is the input of the budget form.
type Budget struct {
	CategoryID string `json:"categoryId" form:"categoryId"`
	Limit      string `json:"limit"      form:"limit"`
}

// BudgetSchema validates Budget.
var BudgetSchema = z.Struct[Budget](z.Shape{
	"categoryId": z.String().Trim().Min(1).Max(20),
	"limit":      z.String().Trim().Min(1).Max(30),
})

// Invite is the input of the invite form.
type Invite struct {
	Email string `json:"email" form:"email"`
	Role  string `json:"role"  form:"role"`
}

// InviteSchema validates Invite.
var InviteSchema = z.Struct[Invite](z.Shape{
	"email": z.String().Trim().Min(3).Max(254),
	"role":  z.String().Trim().Min(1).Max(20),
})

// Workspace is the input of the workspace settings form.
type Workspace struct {
	Name     string `json:"name"     form:"name"`
	Currency string `json:"currency" form:"currency"`
	// Timezone is an IANA name; empty keeps the current one.
	Timezone string `json:"timezone" form:"timezone"`
}

// WorkspaceSchema validates Workspace.
var WorkspaceSchema = z.Struct[Workspace](z.Shape{
	"name":     z.String().Trim().Min(1).Max(80),
	"currency": z.String().Trim().Min(3).Max(3),
	"timezone": z.String().Trim().Max(64).Optional(),
})

// ImportMapping is the input of the column mapping step of a CSV import.
type ImportMapping struct {
	Date        string `json:"date"        form:"date"`
	Description string `json:"description" form:"description"`
	Amount      string `json:"amount"      form:"amount"`
	Category    string `json:"category"    form:"category"`
	DateFormat  string `json:"dateFormat"  form:"dateFormat"`
	Invert      string `json:"invert"      form:"invert"`
}

// ImportMappingSchema validates ImportMapping.
var ImportMappingSchema = z.Struct[ImportMapping](z.Shape{
	"date":        z.String().Trim().Min(1).Max(120),
	"description": z.String().Trim().Min(1).Max(120),
	"amount":      z.String().Trim().Min(1).Max(120),
	"category":    z.String().Trim().Max(120).Optional(),
	"dateFormat":  z.String().Trim().Max(20).Optional(),
	"invert":      z.String().Trim().Max(5).Optional(),
})

// MemberRole is the input of the change-role form.
type MemberRole struct {
	Role string `json:"role" form:"role"`
}

// MemberRoleSchema validates MemberRole.
var MemberRoleSchema = z.Struct[MemberRole](z.Shape{
	"role": z.String().Trim().Min(1).Max(20),
})

// Transfer is the input of the ownership transfer form.
type Transfer struct {
	UserID string `json:"userId" form:"userId"`
}

// TransferSchema validates Transfer.
var TransferSchema = z.Struct[Transfer](z.Shape{
	"userId": z.String().Trim().Min(1).Max(20),
})

// AIKey is the input of the AI key form. The key field is named apiSecret so
// the framework never flashes it back as old input after a validation error
// (its secret-field rule matches "secret").
type AIKey struct {
	Provider  string `json:"provider"  form:"provider"`
	APISecret string `json:"apiSecret" form:"apiSecret"`
	Label     string `json:"label"     form:"label"`
	BaseURL   string `json:"baseUrl"   form:"baseUrl"`
	Model     string `json:"model"     form:"model"`
}

// AIKeySchema validates AIKey.
var AIKeySchema = z.Struct[AIKey](z.Shape{
	"provider":  z.String().Trim().Min(1).Max(30),
	"apiSecret": z.String().Trim().Min(1).Max(512),
	"label":     z.String().Trim().Max(80).Optional(),
	"baseUrl":   z.String().Trim().Max(500).Optional(),
	"model":     z.String().Trim().Max(100).Optional(),
})

// AISettings is the input of the AI switches form.
type AISettings struct {
	Enabled    bool   `json:"enabled"    form:"enabled"`
	Categorize bool   `json:"categorize" form:"categorize"`
	Assistant  bool   `json:"assistant"  form:"assistant"`
	Limit      string `json:"limit"      form:"limit"`
	Consent    bool   `json:"consent"    form:"consent"`
}

// AISettingsSchema validates AISettings.
var AISettingsSchema = z.Struct[AISettings](z.Shape{
	"enabled":    z.Bool().Optional(),
	"categorize": z.Bool().Optional(),
	"assistant":  z.Bool().Optional(),
	"limit":      z.String().Trim().Min(1).Max(9),
	"consent":    z.Bool().Optional(),
})

// AccountTransfer is the input of the transfer form: money moves from one
// account to another of the same workspace.
type AccountTransfer struct {
	FromAccountID string `json:"fromAccountId" form:"fromAccountId"`
	ToAccountID   string `json:"toAccountId"   form:"toAccountId"`
	Amount        string `json:"amount"        form:"amount"`
	Description   string `json:"description"   form:"description"`
	Date          string `json:"date"          form:"date"`
}

// AccountTransferSchema validates AccountTransfer.
var AccountTransferSchema = z.Struct[AccountTransfer](z.Shape{
	"fromAccountId": z.String().Trim().Min(1).Max(20),
	"toAccountId":   z.String().Trim().Min(1).Max(20),
	"amount":        z.String().Trim().Min(1).Max(30),
	"description":   z.String().Trim().Max(200).Optional(),
	"date":          z.String().Trim().Min(10).Max(10),
})

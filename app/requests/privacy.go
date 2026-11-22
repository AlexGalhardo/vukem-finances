package requests

import "github.com/AlexGalhardo/vukem/z"

// DeleteWorkspace confirms the deletion of the current workspace: its exact name
// and the password of the person asking.
type DeleteWorkspace struct {
	Name     string `json:"name"     form:"name"`
	Password string `json:"password" form:"password"`
}

// DeleteWorkspaceSchema validates DeleteWorkspace. The name is not trimmed: the
// person types it exactly.
var DeleteWorkspaceSchema = z.Struct[DeleteWorkspace](z.Shape{
	"name":     z.String().Min(1).Max(200),
	"password": z.String().Min(1).Max(256),
})

// EraseLocal confirms "erase all local data": the typed word and the password.
type EraseLocal struct {
	Confirmation string `json:"confirmation" form:"confirmation"`
	Password     string `json:"password"     form:"password"`
}

// EraseLocalSchema validates EraseLocal.
var EraseLocalSchema = z.Struct[EraseLocal](z.Shape{
	"confirmation": z.String().Min(1).Max(50),
	"password":     z.String().Min(1).Max(256),
})

// DeleteAccount confirms the deletion of the account: the typed confirmation and
// the password.
type DeleteAccount struct {
	Confirmation string `json:"confirmation" form:"confirmation"`
	Password     string `json:"password"     form:"password"`
}

// DeleteAccountSchema validates DeleteAccount.
var DeleteAccountSchema = z.Struct[DeleteAccount](z.Shape{
	"confirmation": z.String().Min(1).Max(200),
	"password":     z.String().Min(1).Max(256),
})

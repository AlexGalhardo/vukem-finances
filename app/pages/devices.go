package pages

import "github.com/AlexGalhardo/vukem/inertia"

// Devices lists the phones that sync the current workspace for the person.
var Devices = inertia.Define[DevicesProps]("Devices")

// DeviceView is a device as the page shows it. Status is live or revoked.
type DeviceView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	CreatedAt   string `json:"createdAt"`
	LastSeenAt  string `json:"lastSeenAt"`
	LastPullSeq int64  `json:"lastPullSeq"`
	RevokedAt   string `json:"revokedAt"`
	Status      string `json:"status"`
}

// CreatedPairing is the one time answer of "Add device": the code to type on
// the phone and the address to type it with.
type CreatedPairing struct {
	Code      string `json:"code"`
	Server    string `json:"server"`
	ExpiresAt string `json:"expiresAt"`
}

// DevicesProps are the props of Devices. Pairing is set only on the response
// that makes a code, and never again.
type DevicesProps struct {
	Devices []DeviceView    `json:"devices"`
	Pairing *CreatedPairing `json:"pairing"`
}

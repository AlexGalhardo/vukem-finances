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

// DevicesProps are the props of Devices.
type DevicesProps struct {
	Devices []DeviceView `json:"devices"`
}

const dict = {
  devices: "Devices",
  devicesIntro:
    "The phones that sync this workspace with the server. A phone registers itself with an API token that holds the sync scope.",
  devicesListTitle: "Your devices in this workspace",
  devicesNone: "No device syncs this workspace yet.",
  devicesName: "Name",
  devicesPlatform: "Platform",
  devicesLastSeen: "Last seen",
  devicesLastPull: "Last pull (change number)",
  devicesNeverSeen: "never",
  devicesStatus: "Status",
  devicesStatusLive: "Syncing",
  devicesStatusRevoked: "Revoked",
  devicesRevoke: "Revoke",
  devicesRevokeNote:
    "Revoking a device stops its sync at once. The device keeps its local data until you clear it on the device itself. Revoking an API token revokes its devices too.",
  devicesAdd: "Add device",
  devicesPairTitle: "Pair a phone",
  devicesPairBody:
    "In the app, open Connect and type the server address and this code. The code works once and expires in 10 minutes.",
  devicesPairServer: "Server address",
  devicesPairCode: "Pairing code",
  devicesPairExpires: "Expires at",
  devicesPairCopy: "Copy code",
  devicesPairDismiss: "Done",
} as const;
export default dict;

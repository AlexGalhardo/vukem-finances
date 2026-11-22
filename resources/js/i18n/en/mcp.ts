const dict = {
  connectedApps: "Connected apps",
  connectedAppsIntro:
    "AI clients you let work in this workspace through MCP. Each one can do only what you approved, never more than your role allows, and you can disconnect it at any time.",
  connectedAppsListTitle: "Your connected apps in this workspace",
  connectedAppsNone: "No AI client is connected to this workspace.",
  connectedAppsClient: "Client",
  connectedAppsScopes: "Allowed to",
  connectedAppsLastUsed: "Last used",
  connectedAppsNeverUsed: "never",
  connectedAppsStatus: "Status",
  connectedAppsStatusLive: "Connected",
  connectedAppsStatusRevoked: "Disconnected",
  connectedAppsRevoke: "Disconnect",
  connectedAppsRevokeNote:
    "Disconnecting stops the client at once: its next call is refused, and it cannot get a new token.",
  consentTitle: "Connect an AI client",
  consentIntro: "{client} asks to work with your finances.",
  consentClient: "Client",
  consentClientId: "Verified identity",
  consentRedirect: "Returns to",
  consentWorkspace: "Workspace",
  consentWorkspacePick: "Use another workspace",
  consentRole: "Your role here: {role}",
  consentScopes: "It will be allowed to",
  consentScopeWrites: "changes data",
  consentScopeNotAllowed: "not available with your role",
  consentApprove: "Approve",
  consentDeny: "Deny",
  consentNote:
    "The client acts as you in this workspace only. It can never do more than your role allows, and you can disconnect it under Connected apps.",
} as const;
export default dict;

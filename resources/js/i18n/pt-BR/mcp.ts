import type en from "../en/mcp";

const dict = {
  connectedApps: "Apps conectados",
  connectedAppsIntro:
    "Clientes de IA que você deixa trabalhar neste workspace por MCP. Cada um faz só o que você aprovou, nunca mais do que o seu papel permite, e você pode desconectá-lo quando quiser.",
  connectedAppsListTitle: "Seus apps conectados neste workspace",
  connectedAppsNone: "Nenhum cliente de IA está conectado a este workspace.",
  connectedAppsClient: "Cliente",
  connectedAppsScopes: "Pode",
  connectedAppsLastUsed: "Último uso",
  connectedAppsNeverUsed: "nunca",
  connectedAppsStatus: "Situação",
  connectedAppsStatusLive: "Conectado",
  connectedAppsStatusRevoked: "Desconectado",
  connectedAppsRevoke: "Desconectar",
  connectedAppsRevokeNote:
    "Desconectar para o cliente na hora: a próxima chamada é recusada e ele não consegue um novo token.",
  consentTitle: "Conectar um cliente de IA",
  consentIntro: "{client} pede para trabalhar com as suas finanças.",
  consentClient: "Cliente",
  consentClaimedName: "Nome que ele diz ter (não verificado)",
  consentClientId: "Identidade verificada",
  consentRedirect: "Volta para",
  consentWorkspace: "Workspace",
  consentWorkspacePick: "Usar outro workspace",
  consentRole: "Seu papel aqui: {role}",
  consentScopes: "Ele poderá",
  consentScopeWrites: "altera dados",
  consentScopeNotAllowed: "indisponível com o seu papel",
  consentApprove: "Aprovar",
  consentDeny: "Negar",
  consentNote:
    "O cliente age como você somente neste workspace. Ele nunca faz mais do que o seu papel permite, e você pode desconectá-lo em Apps conectados.",
} satisfies Record<keyof typeof en, string>;
export default dict;

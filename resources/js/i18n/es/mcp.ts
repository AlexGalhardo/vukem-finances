import type en from "../en/mcp";

const dict = {
  connectedApps: "Apps conectadas",
  connectedAppsIntro:
    "Clientes de IA que dejas trabajar en este espacio de trabajo por MCP. Cada uno hace solo lo que aprobaste, nunca más de lo que permite tu rol, y puedes desconectarlo cuando quieras.",
  connectedAppsListTitle: "Tus apps conectadas en este espacio de trabajo",
  connectedAppsNone: "Ningún cliente de IA está conectado a este espacio de trabajo.",
  connectedAppsClient: "Cliente",
  connectedAppsScopes: "Puede",
  connectedAppsLastUsed: "Último uso",
  connectedAppsNeverUsed: "nunca",
  connectedAppsStatus: "Estado",
  connectedAppsStatusLive: "Conectada",
  connectedAppsStatusRevoked: "Desconectada",
  connectedAppsRevoke: "Desconectar",
  connectedAppsRevokeNote:
    "Desconectar detiene al cliente al instante: su próxima llamada se rechaza y no puede obtener un token nuevo.",
  consentTitle: "Conectar un cliente de IA",
  consentIntro: "{client} pide trabajar con tus finanzas.",
  consentClient: "Cliente",
  consentClientId: "Identidad verificada",
  consentRedirect: "Vuelve a",
  consentWorkspace: "Espacio de trabajo",
  consentWorkspacePick: "Usar otro espacio de trabajo",
  consentRole: "Tu rol aquí: {role}",
  consentScopes: "Podrá",
  consentScopeWrites: "cambia datos",
  consentScopeNotAllowed: "no disponible con tu rol",
  consentApprove: "Aprobar",
  consentDeny: "Denegar",
  consentNote:
    "El cliente actúa como tú solo en este espacio de trabajo. Nunca puede hacer más de lo que permite tu rol, y puedes desconectarlo en Apps conectadas.",
} satisfies Record<keyof typeof en, string>;
export default dict;

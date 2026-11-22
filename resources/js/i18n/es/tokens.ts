import type en from "../en/tokens";

const dict = {
  tokens: "Tokens de API",
  tokensIntro:
    "Un token permite que un programa lea o escriba los datos de este workspace por la API. Hace solo lo que sus alcances permiten, y nunca más de lo que permite tu rol.",
  tokensCreateTitle: "Nuevo token",
  tokensName: "Nombre",
  tokensScopes: "Alcances",
  tokensExpiry: "Expira en",
  tokensDays: "días",
  tokensCreate: "Crear token",
  tokensCreatedTitle: "Copia tu token ahora",
  tokensCreatedBody:
    "Esta es la única vez que se muestra el token. Solo se guarda un hash, así que no se puede mostrar de nuevo.",
  tokensCreatedDismiss: "Ya lo copié",
  tokensCopy: "Copiar",
  tokensListTitle: "Tus tokens en este workspace",
  tokensNone: "No tienes tokens en este workspace.",
  tokensLastUsed: "Último uso",
  tokensNeverUsed: "nunca",
  tokensExpires: "Expira",
  tokensStatus: "Estado",
  tokensStatusLive: "Activo",
  tokensStatusExpired: "Expirado",
  tokensStatusRevoked: "Revocado",
  tokensRevoke: "Revocar",
  tokensLimit: "Puedes tener como máximo {max} tokens activos por workspace.",
  tokensHelpScopes: "Acerca de los alcances",
  tokensHelpExpiry: "Acerca de la caducidad",
  tokensCopied: "Copiado",
  tokensCreateDescription: "Elige un nombre, lo que puede hacer el token y cuándo deja de funcionar.",
  tokensListDescription: "Los tokens que creaste aquí. Revocar uno lo detiene al instante.",
} satisfies Record<keyof typeof en, string>;
export default dict;

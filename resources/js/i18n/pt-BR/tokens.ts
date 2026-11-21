import type en from "../en/tokens";

const dict = {
  tokens: "Tokens de API",
  tokensIntro:
    "Um token permite que um programa leia ou grave os dados deste workspace pela API. Ele faz só o que seus escopos permitem, e nunca mais do que o seu papel permite.",
  tokensCreateTitle: "Novo token",
  tokensName: "Nome",
  tokensScopes: "Escopos",
  tokensExpiry: "Expira em",
  tokensDays: "dias",
  tokensCreate: "Criar token",
  tokensCreatedTitle: "Copie seu token agora",
  tokensCreatedBody:
    "Esta é a única vez que o token aparece. Só um hash é guardado, então ele não pode ser mostrado de novo.",
  tokensCreatedDismiss: "Já copiei",
  tokensCopy: "Copiar",
  tokensListTitle: "Seus tokens neste workspace",
  tokensNone: "Você não tem tokens neste workspace.",
  tokensLastUsed: "Último uso",
  tokensNeverUsed: "nunca",
  tokensExpires: "Expira",
  tokensStatus: "Situação",
  tokensStatusLive: "Ativo",
  tokensStatusExpired: "Expirado",
  tokensStatusRevoked: "Revogado",
  tokensRevoke: "Revogar",
  tokensLimit: "Você pode ter no máximo {max} tokens ativos por workspace.",
} satisfies Record<keyof typeof en, string>;
export default dict;

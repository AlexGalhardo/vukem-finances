import type en from "../en/components";

const dict = {
  pagination: "Paginação",
  previousPage: "Página anterior",
  nextPage: "Próxima página",
  page: "Página",
  loading: "Carregando",
  dismiss: "Dispensar",
  notifications: "Notificações",
  moneyMinus: "menos",
  moneyPlus: "mais",
} satisfies Record<keyof typeof en, string>;
export default dict;

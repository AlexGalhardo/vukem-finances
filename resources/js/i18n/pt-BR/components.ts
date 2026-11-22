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
  showPassword: "Mostrar senha",
  hidePassword: "Ocultar senha",
  passwordMet: "atendido",
  passwordNotMet: "ainda não atendido",
  passwordStrength: "Força",
  passwordAdviceNote: "sugestão, não um requisito",
  confirmTypeName: "Digite o nome para confirmar:",
  chartOther: "Outros",
  confirmNameLabel: "Nome",
} satisfies Record<keyof typeof en, string>;
export default dict;

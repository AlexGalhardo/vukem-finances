import type en from "../en/components";

const dict = {
  pagination: "Paginación",
  previousPage: "Página anterior",
  nextPage: "Página siguiente",
  page: "Página",
  loading: "Cargando",
  dismiss: "Descartar",
  notifications: "Notificaciones",
  moneyMinus: "menos",
  moneyPlus: "más",
  showPassword: "Mostrar contraseña",
  hidePassword: "Ocultar contraseña",
  passwordMet: "cumplido",
  passwordNotMet: "aún no cumplido",
  passwordStrength: "Seguridad",
  passwordAdviceNote: "consejo, no un requisito",
  confirmTypeName: "Escribe el nombre para confirmar:",
  chartOther: "Otros",
  confirmNameLabel: "Nombre",
} satisfies Record<keyof typeof en, string>;
export default dict;

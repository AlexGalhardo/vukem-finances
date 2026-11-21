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
} satisfies Record<keyof typeof en, string>;
export default dict;

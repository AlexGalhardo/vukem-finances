import type en from "../en/shell";

const dict = {
  dashboard: "Resumen",
  transactions: "Movimientos",
  accounts: "Cuentas",
  categories: "Categorías",
  budgets: "Presupuestos",
  import: "Importar extracto",
  workspace: "Espacio de trabajo",
  billing: "Suscripción",
  settings: "Configuración",
  admin: "Administración",
  logout: "Cerrar sesión",
  language: "Idioma",
  light: "Activar tema claro",
  dark: "Activar tema oscuro",
  plan: "Plan",
  navigate: "Navegación",
  personalFinance: "Finanzas personales",
} satisfies Record<keyof typeof en, string>;
export default dict;

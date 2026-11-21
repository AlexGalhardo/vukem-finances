import type en from "../en/shell";

const dict = {
  dashboard: "Visão geral",
  transactions: "Transações",
  accounts: "Contas",
  categories: "Categorias",
  budgets: "Orçamentos",
  import: "Importar extrato",
  workspace: "Espaço de trabalho",
  billing: "Assinatura",
  settings: "Configurações",
  admin: "Administração",
  logout: "Sair",
  language: "Idioma",
  light: "Ativar tema claro",
  dark: "Ativar tema escuro",
  plan: "Plano",
  navigate: "Navegação",
  personalFinance: "Finanças pessoais",
} satisfies Record<keyof typeof en, string>;
export default dict;

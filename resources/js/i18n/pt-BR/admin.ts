import type en from "../en/admin";

const dict = {
  admDashboard: "Painel",
  admPanel: "Admin",
  admNavLabel: "Seções do admin",
  admLoading: "Carregando...",
  admSearch: "Buscar",
  admBack: "Voltar",
  admBackToApp: "Voltar ao app",
  admPrevious: "Anterior",
  admNext: "Próxima",
  admPager: "Página {page} de {last}, {total} registros",
  admAuditLog: "Log de auditoria",
  admFeatureFlags: "Feature flags",
  admRefundTitle: "Marcar este pagamento como reembolsado?",
  admCancel: "Cancelar",
} satisfies Record<keyof typeof en, string>;
export default dict;

import type en from "../en/admin";

const dict = {
  admDashboard: "Panel",
  admPanel: "Admin",
  admNavLabel: "Secciones del admin",
  admLoading: "Cargando...",
  admSearch: "Buscar",
  admBack: "Volver",
  admBackToApp: "Volver a la app",
  admPrevious: "Anterior",
  admNext: "Siguiente",
  admPager: "Página {page} de {last}, {total} registros",
  admAuditLog: "Registro de auditoría",
  admFeatureFlags: "Feature flags",
  admRefundTitle: "¿Marcar este pago como reembolsado?",
  admCancel: "Cancelar",
} satisfies Record<keyof typeof en, string>;
export default dict;

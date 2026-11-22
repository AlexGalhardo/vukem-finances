import type en from "../en/goals";

const dict = {
  goals: "Metas",
  goalsDescription:
    "Aparta dinero para lo que importa. Las aportaciones son anotaciones: nunca mueven dinero entre cuentas.",
  goalsEmpty: "Aún no hay metas",
  goalsEmptyDescription: "Añade un importe, y una fecha si la tienes, y registra lo que apartas.",
  goalNew: "Nueva meta",
  goalName: "Nombre",
  goalTarget: "Importe de la meta",
  goalTargetDate: "Fecha de la meta (opcional)",
  goalSave: "Guardar meta",
  goalSaved: "{saved} de {target}",
  goalProgress: "Progreso de {name}",
  goalRemaining: "Faltan {amount}",
  goalReached: "Meta alcanzada",
  goalRequired: "Aparta {amount} al mes durante {n} mes(es)",
  goalOverdue: "La fecha de la meta ya pasó",
  goalProjected: "Con tu ritmo reciente la alcanzas hacia el {date}",
  goalContribute: "Añadir una aportación",
  goalAmount: "Importe",
  goalAmountHelp: "Usa un importe negativo para retirar dinero.",
  goalDate: "Fecha",
  goalNote: "Nota (opcional)",
  goalContributionAdd: "Añadir aportación",
  goalContributions: "Aportaciones recientes",
  goalArchive: "Archivar",
  goalUnarchive: "Desarchivar",
  goalArchived: "Archivada",
  goalEdit: "Editar meta",
  goalNoContributions: "Aún no hay aportaciones",
} satisfies Record<keyof typeof en, string>;
export default dict;

import type en from "../en/search";

const dict = {
  searchTransactions: "Buscar transacciones",
  searchLabel: "Buscar en las descripciones",
  searchPlaceholder: "Escribe para buscar, / para venir aquí",
  filters: "Filtros",
  applyFilters: "Aplicar filtros",
  minAmount: "Importe mínimo",
  maxAmount: "Importe máximo",
  tag: "Etiqueta",
  tags: "Etiquetas",
  tagsHelp: "Escribe una etiqueta y pulsa Enter. Hasta 10. Las nuevas se crean al guardar.",
  tagAdded: "Etiqueta añadida",
  tagRemoved: "Etiqueta quitada",
  tagsFull: "Una transacción admite como máximo 10 etiquetas.",
  removeTag: "Quitar etiqueta",
} as const satisfies Record<keyof typeof en, string>;
export default dict;

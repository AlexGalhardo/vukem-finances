import type en from "../en/search";

const dict = {
  searchTransactions: "Buscar transações",
  searchLabel: "Buscar nas descrições",
  searchPlaceholder: "Digite para buscar, / para vir aqui",
  filters: "Filtros",
  applyFilters: "Aplicar filtros",
  minAmount: "Valor mínimo",
  maxAmount: "Valor máximo",
  tag: "Tag",
  tags: "Tags",
  tagsHelp: "Digite uma tag e pressione Enter. Até 10. Tags novas são criadas ao salvar.",
  tagAdded: "Tag adicionada",
  tagRemoved: "Tag removida",
  tagsFull: "Uma transação aceita no máximo 10 tags.",
  removeTag: "Remover tag",
} as const satisfies Record<keyof typeof en, string>;
export default dict;

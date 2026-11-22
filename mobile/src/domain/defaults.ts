import type { Locale } from "../i18n";

interface DefaultCategory {
  kind: "income" | "expense";
  color: string;
  names: Record<Locale, string>;
}

/** The starter categories of the first run, named in the person's language (stored as typed, like any other). */
export const DEFAULT_CATEGORIES: readonly DefaultCategory[] = [
  { kind: "expense", color: "green", names: { en: "Groceries", "pt-BR": "Mercado", es: "Supermercado" } },
  { kind: "expense", color: "orange", names: { en: "Dining out", "pt-BR": "Restaurantes", es: "Restaurantes" } },
  { kind: "expense", color: "blue", names: { en: "Transport", "pt-BR": "Transporte", es: "Transporte" } },
  { kind: "expense", color: "slate", names: { en: "Housing", "pt-BR": "Moradia", es: "Vivienda" } },
  { kind: "expense", color: "red", names: { en: "Health", "pt-BR": "Saúde", es: "Salud" } },
  { kind: "expense", color: "violet", names: { en: "Leisure", "pt-BR": "Lazer", es: "Ocio" } },
  { kind: "expense", color: "teal", names: { en: "Bills", "pt-BR": "Contas", es: "Facturas" } },
  { kind: "income", color: "green", names: { en: "Salary", "pt-BR": "Salário", es: "Salario" } },
  { kind: "income", color: "olive", names: { en: "Other income", "pt-BR": "Outras receitas", es: "Otros ingresos" } },
];

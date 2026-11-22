import type en from "../en/budgets";

const dict = {
  budgetsProOnly: "Los presupuestos son una función Pro.",
  budgetsProTrial: "Empieza la prueba de 14 días",
  budgetsEmpty: "Aún no hay presupuestos. Recibes una alerta cuando un presupuesto llega al 80% y al 100%.",
  budgetsOverall: "General",
  budgetsOverallHelp: "Todos los gastos del mes, de cualquier categoría. Las transferencias nunca cuentan.",
  budgetsSpent: "{spent} de {limit}",
  budgetsCarry: "Incluye {amount} que sobraron del mes pasado",
  budgetsStatusOk: "Dentro de lo previsto",
  budgetsStatusNear: "Cerca del límite",
  budgetsStatusOver: "Por encima del presupuesto",
  budgetsProgress: "Gastado en {name}",
  budgetsAllSpending: "General (todos los gastos)",
  budgetsRollover: "Llevar lo que sobró del mes pasado",
  budgetsRolloverHelp:
    "Lo que el mes pasado dejó sin gastar se suma a este mes, nunca una cantidad negativa. No se acumula: cuenta solo el mes anterior.",
  budgetsSave: "Guardar presupuesto",
  budgetsPreviousMonth: "Mes anterior",
  budgetsNextMonth: "Mes siguiente",
} satisfies Record<keyof typeof en, string>;
export default dict;

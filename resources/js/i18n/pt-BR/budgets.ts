import type en from "../en/budgets";

const dict = {
  budgetsProOnly: "Orçamentos são um recurso Pro.",
  budgetsProTrial: "Comece o teste de 14 dias",
  budgetsEmpty: "Nenhum orçamento ainda. Você recebe um alerta quando um orçamento chega a 80% e a 100%.",
  budgetsOverall: "Geral",
  budgetsOverallHelp: "Todas as despesas do mês, de qualquer categoria. Transferências nunca contam.",
  budgetsSpent: "{spent} de {limit}",
  budgetsCarry: "Inclui {amount} que sobraram do mês passado",
  budgetsStatusOk: "Dentro do previsto",
  budgetsStatusNear: "Perto do limite",
  budgetsStatusOver: "Acima do orçamento",
  budgetsProgress: "Gasto em {name}",
  budgetsAllSpending: "Geral (todos os gastos)",
  budgetsRollover: "Levar o que sobrou do mês passado",
  budgetsRolloverHelp:
    "O que o mês passado deixou sem gastar é somado a este mês, nunca um valor negativo. Não acumula: conta só o mês anterior.",
  budgetsSave: "Salvar orçamento",
  budgetsPreviousMonth: "Mês anterior",
  budgetsNextMonth: "Próximo mês",
} satisfies Record<keyof typeof en, string>;
export default dict;

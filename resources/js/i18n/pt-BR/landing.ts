import type en from "../en/landing";

const dict = {
  landExample: "Dados de exemplo",
  landMonth: "Neste mês",
  landRowSalary: "Salário",
  landRowRent: "Aluguel",
  landRowGroceries: "Mercado",
  landRowSavings: "Enviado à reserva",
  landCatIncome: "Receita",
  landCatHome: "Moradia",
  landCatFood: "Alimentação",
  landCatTransfer: "Transferência",
  landFeature1Body: "Banco, dinheiro e cartões lado a lado, com um total em que você pode confiar.",
  landFeature2Body: "Orçamentos e metas que mostram até onde você foi, em números e não só em barras.",
  landFeature3Body: "Divida um workspace com quem você divide as contas, cada pessoa com seu papel.",
  landFeaturesLabel: "O que você recebe",
  landBars: "Fluxo de caixa dos últimos doze meses",
  landPlansLabel: "Planos",
} satisfies Record<keyof typeof en, string>;
export default dict;

import type en from "../en/transactions";

const dict = {
  newExpense: "Nova despesa",
  newIncome: "Nova receita",
  newTransfer: "Nova transferência",
  editTransfer: "Editar transferência",
  addTransfer: "Adicionar transferência",
  saveTransfer: "Salvar transferência",
  transfer: "Transferência",
  transferDescription: "Mova dinheiro entre suas contas. Uma transferência nunca é receita nem despesa.",
  quickAddDescription: "Digite o valor e pressione Enter para salvar ou Escape para fechar.",
  fromAccount: "Conta de origem",
  toAccount: "Conta de destino",
  transferTo: "Transferência para",
  transferFrom: "Transferência de",
  quickAddLoadError: "Não foi possível carregar suas contas. Feche esta janela e tente de novo.",
  quickAddSaveError: "Não foi possível salvar. Verifique a conexão e tente de novo.",
  savedTransaction: "Transação adicionada.",
  savedTransfer: "Transferência adicionada.",
  updatedTransfer: "Transferência atualizada.",
  noAccountsToPick: "Adicione primeiro uma conta que não esteja arquivada.",
  needTwoAccounts: "Uma transferência precisa de duas contas que não estejam arquivadas.",
  transferLegLocked: "Este é um lado de uma transferência. Edite a transferência para alterar os dois lados.",
} satisfies Record<keyof typeof en, string>;
export default dict;

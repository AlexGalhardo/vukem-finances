import type en from "../en/accounts";

const dict = {
  accountsDescription: "Cada lugar onde seu dinheiro está, com o saldo real.",
  accountsTable: "Contas e seus saldos",
  netWorth: "Patrimônio",
  netWorthNote: "A soma das contas que não estão arquivadas.",
  accountBalance: "Saldo",
  showArchived: "Mostrar arquivadas",
  archived: "Arquivada",
  archive: "Arquivar",
  unarchive: "Desarquivar",
  editAccount: "Editar conta",
  saveAccount: "Salvar conta",
  investment: "Investimento",
  wallet: "Carteira",
  emptyAccounts: "Nenhuma conta para mostrar",
  emptyAccountsDescription: "Adicione uma conta para acompanhar o saldo dela.",
  openingHelp: "O saldo antes da primeira transação que você registrar.",
  archiveHint: "Arquivar esconde a conta dos seletores e mantém o histórico e o saldo.",
  confirmDeleteAccount:
    "Excluir uma conta também remove as transações dela e o outro lado das transferências. Arquivar mantém o histórico.",
  transactionsWord: "transações",
} satisfies Record<keyof typeof en, string>;
export default dict;

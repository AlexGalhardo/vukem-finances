import type en from "../en/accounts";

const dict = {
  accountsPlanLimit: "El plan {plan} permite {count} cuentas.",
  accountsUpgrade: "Mejora tu plan para tener más",
  accountsDescription: "Cada lugar donde está tu dinero, con su saldo real.",
  accountsTable: "Cuentas y sus saldos",
  netWorth: "Patrimonio",
  netWorthNote: "La suma de las cuentas que no están archivadas.",
  accountBalance: "Saldo",
  showArchived: "Mostrar archivadas",
  archived: "Archivada",
  archive: "Archivar",
  unarchive: "Desarchivar",
  editAccount: "Editar cuenta",
  saveAccount: "Guardar cuenta",
  investment: "Inversión",
  wallet: "Billetera",
  emptyAccounts: "No hay cuentas que mostrar",
  emptyAccountsDescription: "Añade una cuenta para seguir su saldo.",
  openingHelp: "El saldo antes de la primera transacción que registres.",
  archiveHint: "Archivar oculta la cuenta de los selectores y conserva su historial y saldo.",
  confirmDeleteAccount:
    "Eliminar una cuenta también elimina sus transacciones y el otro lado de sus transferencias. Archivar conserva el historial.",
  transactionsWord: "transacciones",
} satisfies Record<keyof typeof en, string>;
export default dict;

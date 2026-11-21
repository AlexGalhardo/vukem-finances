import type en from "../en/transactions";

const dict = {
  newExpense: "Nuevo gasto",
  newIncome: "Nuevo ingreso",
  newTransfer: "Nueva transferencia",
  editTransfer: "Editar transferencia",
  addTransfer: "Añadir transferencia",
  saveTransfer: "Guardar transferencia",
  transfer: "Transferencia",
  transferDescription: "Mueve dinero entre tus cuentas. Una transferencia nunca es ingreso ni gasto.",
  quickAddDescription: "Escribe el importe y pulsa Enter para guardar o Escape para cerrar.",
  fromAccount: "Cuenta de origen",
  toAccount: "Cuenta de destino",
  transferTo: "Transferencia a",
  transferFrom: "Transferencia de",
  quickAddLoadError: "No se pudieron cargar tus cuentas. Cierra esta ventana e inténtalo de nuevo.",
  quickAddSaveError: "No se pudo guardar. Revisa la conexión e inténtalo de nuevo.",
  savedTransaction: "Transacción añadida.",
  savedTransfer: "Transferencia añadida.",
  updatedTransfer: "Transferencia actualizada.",
  noAccountsToPick: "Primero añade una cuenta que no esté archivada.",
  needTwoAccounts: "Una transferencia necesita dos cuentas que no estén archivadas.",
  transferLegLocked: "Este es un lado de una transferencia. Edita la transferencia para cambiar ambos lados.",
} satisfies Record<keyof typeof en, string>;
export default dict;

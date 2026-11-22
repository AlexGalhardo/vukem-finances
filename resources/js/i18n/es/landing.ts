import type en from "../en/landing";

const dict = {
  landExample: "Datos de ejemplo",
  landMonth: "Este mes",
  landRowSalary: "Salario",
  landRowRent: "Alquiler",
  landRowGroceries: "Supermercado",
  landRowSavings: "Enviado a ahorros",
  landCatIncome: "Ingreso",
  landCatHome: "Vivienda",
  landCatFood: "Comida",
  landCatTransfer: "Transferencia",
  landFeature1Body: "Banco, efectivo y tarjetas lado a lado, con un total en el que puedes confiar.",
  landFeature2Body: "Presupuestos y metas que muestran cuánto avanzaste, en números y no solo en barras.",
  landFeature3Body: "Comparte un espacio con quienes compartes gastos, cada persona con su rol.",
  landFeaturesLabel: "Lo que obtienes",
  landBars: "Flujo de caja de los últimos doce meses",
  landPlansLabel: "Planes",
} satisfies Record<keyof typeof en, string>;
export default dict;

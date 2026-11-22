import type en from "../en/goals";

const dict = {
  goals: "Metas",
  goalsDescription:
    "Separe dinheiro para o que importa. As contribuições são anotações: nunca movem dinheiro entre contas.",
  goalsEmpty: "Nenhuma meta ainda",
  goalsEmptyDescription: "Cadastre um valor, e uma data se tiver, e registre o que você separou.",
  goalNew: "Nova meta",
  goalName: "Nome",
  goalTarget: "Valor da meta",
  goalTargetDate: "Data da meta (opcional)",
  goalSave: "Salvar meta",
  goalSaved: "{saved} de {target}",
  goalProgress: "Progresso de {name}",
  goalRemaining: "Faltam {amount}",
  goalReached: "Meta alcançada",
  goalRequired: "Separe {amount} por mês durante {n} mês(es)",
  goalOverdue: "A data da meta já passou",
  goalProjected: "No seu ritmo recente você chega lá por volta de {date}",
  goalContribute: "Adicionar uma contribuição",
  goalAmount: "Valor",
  goalAmountHelp: "Use um valor negativo para retirar dinheiro.",
  goalDate: "Data",
  goalNote: "Observação (opcional)",
  goalContributionAdd: "Adicionar contribuição",
  goalContributions: "Contribuições recentes",
  goalArchive: "Arquivar",
  goalUnarchive: "Desarquivar",
  goalArchived: "Arquivada",
  goalEdit: "Editar meta",
  goalNoContributions: "Nenhuma contribuição ainda",
} satisfies Record<keyof typeof en, string>;
export default dict;

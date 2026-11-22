import type en from "../en/privacy";

const dict = {
  privacy: "Privacidade e dados",
  privacyIntro: "Leve seus dados com você, ou apague-os. Cada ação pede sua senha antes.",
  privacyPassword: "Sua senha",
  privacyPasswordHint: "Entrou com o Google e nunca escolheu uma senha? Redefina-a na tela de entrada antes.",
  privacyExportTitle: "Exportar meus dados",
  privacyExportBody:
    "Um arquivo JSON com seu perfil, suas participações, a lista das suas sessões e tokens (nunca os valores) e tudo dos workspaces que você possui. Senhas, segredos de dois fatores, tokens e chaves de IA nunca entram.",
  privacyExportButton: "Exportar meus dados",
  privacyExportStarted: "O download começou. Se não começou, confirme a senha de novo.",
  privacyWorkspaceTitle: "Excluir este workspace",
  privacyWorkspaceBody:
    "Só o dono pode fazer isso. O workspace fica somente leitura para todos agora, os membros são avisados por e-mail e tudo nele é apagado de vez depois de {days} dias. O dono pode cancelar até lá. A assinatura é cancelada na hora; não há reembolso automático.",
  privacyWorkspaceName: "Digite o nome do workspace para confirmar",
  privacyWorkspaceDelete: "Excluir o workspace",
  privacyWorkspaceOwnerOnly: "Só o dono deste workspace pode excluí-lo.",
  privacyWorkspacePersonal:
    "A edição pessoal tem um workspace. Para removê-lo com tudo o que há nele, apague todos os dados locais abaixo.",
  privacyPendingTitle: "Este workspace está marcado para exclusão",
  privacyPendingBody:
    "Ele está somente leitura e será apagado de vez em {date}. Até lá você ainda pode ler e exportar seus dados.",
  privacyPendingCancel: "Cancelar a exclusão",
  privacyPendingOwnerOnly: "Só o dono pode cancelar a exclusão.",
  privacyAccountTitle: "Excluir minha conta",
  privacyAccountBody:
    "Todas as sessões e tokens terminam agora e a conta é apagada de vez depois de {days} dias; entre antes disso para cancelar. Workspaces em que você é o único membro vão junto. Se você é o único dono de um workspace que outras pessoas usam, transfira a propriedade ou exclua esse workspace antes. Nos workspaces que ficam, seus dados privados (conversas do assistente, notificações, tokens, dispositivos) são apagados e seu nome sai do que você escreveu.",
  privacyAccountConfirm: "Digite seu e-mail para confirmar",
  privacyAccountDelete: "Excluir minha conta",
  accountDeletionTitle: "Sua conta está marcada para exclusão",
  accountDeletionBody:
    "Ela será apagada de vez em {date}. Entrar não cancela a exclusão. Suas sessões, tokens e apps conectados foram encerrados e não voltam.",
  accountDeletionCancel: "Manter minha conta",
  accountDeletionSignOut: "Sair",
  privacyEraseTitle: "Apagar todos os dados locais",
  privacyEraseBody:
    "Remove deste dispositivo o banco de dados desta instalação, com tudo o que há nele. Nada é enviado a lugar nenhum e não é preciso rede. O app reinicia vazio, na configuração inicial. Exporte seus dados antes se quiser guardá-los.",
  privacyEraseConfirm: "Digite ERASE para confirmar",
  privacyEraseButton: "Apagar todos os dados locais",
  privacyErased:
    "Os dados locais foram apagados. O app está reiniciando vazio: recarregue esta página em alguns segundos.",
} satisfies Record<keyof typeof en, string>;
export default dict;

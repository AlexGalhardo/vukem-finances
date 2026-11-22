import type en from "../en/auth";

const dict = {
  authErrorSummary: "Confira os campos destacados e tente de novo.",
  authCheckEmailTitle: "Verifique seu e-mail",
  authCheckEmailBody:
    "Se o endereço puder ser usado, enviamos a ele um link para confirmá-lo. Se ele já tiver uma conta, enviamos um link para entrar ou redefinir a senha.",
  authGoToLogin: "Ir para a página de login",
  authVerifyTitle: "Confirme seu e-mail",
  authVerifyBody: "Enviamos um link para {email}. Abra-o para confirmar seu endereço.",
  authVerifyResend: "Enviar o link de novo",
  authForgotTitle: "Esqueceu sua senha?",
  authForgotDescription: "Enviaremos por e-mail um link para você escolher uma nova.",
  authForgotSubmit: "Enviar link de redefinição",
  authBackToLogin: "Voltar ao login",
  authResetTitle: "Escolha uma nova senha",
  authResetDescription: "Use uma senha que você não usa em nenhum outro lugar.",
  authNewPassword: "Nova senha",
  authResetSubmit: "Redefinir senha",
  authTwoFactorTitle: "Autenticação em dois fatores",
  authTwoFactorAppPrompt: "Digite o código do seu app autenticador.",
  authTwoFactorRecoveryPrompt: "Digite um dos seus códigos de recuperação.",
  authTwoFactorCode: "Código",
  authTwoFactorRecoveryCode: "Código de recuperação",
  authTwoFactorUseApp: "Usar um código do app autenticador",
  authTwoFactorUseRecovery: "Perdeu o aparelho? Use um código de recuperação",
  authInviteTitle: "Entre em um espaço de trabalho",
  authInviteBody:
    "Aceite o convite para começar a trabalhar no espaço compartilhado. Ele só funciona para o endereço ao qual foi enviado.",
  authInviteAccept: "Aceitar o convite",
  authErrorBack: "Voltar à página inicial",
} satisfies Record<keyof typeof en, string>;
export default dict;

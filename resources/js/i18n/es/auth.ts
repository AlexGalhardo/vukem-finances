import type en from "../en/auth";

const dict = {
  authErrorSummary: "Revisa los campos resaltados e inténtalo de nuevo.",
  authCheckEmailTitle: "Revisa tu correo",
  authCheckEmailBody:
    "Si la dirección se puede usar, le enviamos un enlace para verificarla. Si ya tiene una cuenta, le enviamos un enlace para entrar o restablecer la contraseña.",
  authGoToLogin: "Ir a la página de inicio de sesión",
  authVerifyTitle: "Verifica tu correo",
  authVerifyBody: "Enviamos un enlace a {email}. Ábrelo para verificar tu dirección.",
  authVerifyResend: "Enviar el enlace de nuevo",
  authForgotTitle: "¿Olvidaste tu contraseña?",
  authForgotDescription: "Te enviaremos por correo un enlace para elegir una nueva.",
  authForgotSubmit: "Enviar enlace de restablecimiento",
  authBackToLogin: "Volver a iniciar sesión",
  authResetTitle: "Elige una nueva contraseña",
  authResetDescription: "Usa una contraseña que no uses en ningún otro sitio.",
  authNewPassword: "Nueva contraseña",
  authResetSubmit: "Restablecer contraseña",
  authTwoFactorTitle: "Autenticación en dos pasos",
  authTwoFactorAppPrompt: "Introduce el código de tu app de autenticación.",
  authTwoFactorRecoveryPrompt: "Introduce uno de tus códigos de recuperación.",
  authTwoFactorCode: "Código",
  authTwoFactorRecoveryCode: "Código de recuperación",
  authTwoFactorUseApp: "Usar un código de la app de autenticación",
  authTwoFactorUseRecovery: "¿Perdiste el dispositivo? Usa un código de recuperación",
  authInviteTitle: "Únete a un espacio de trabajo",
  authInviteBody:
    "Acepta la invitación para empezar a trabajar en el espacio compartido. Solo funciona con la dirección a la que se envió.",
  authInviteAccept: "Aceptar la invitación",
  authErrorBack: "Volver a la página de inicio",
} satisfies Record<keyof typeof en, string>;
export default dict;

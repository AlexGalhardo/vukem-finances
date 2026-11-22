import type en from "../en/privacy";

const dict = {
  privacy: "Privacidad y datos",
  privacyIntro: "Llévate tus datos, o bórralos. Cada acción pide tu contraseña antes.",
  privacyPassword: "Tu contraseña",
  privacyPasswordHint:
    "¿Entraste con Google y nunca elegiste una contraseña? Restablécela antes desde la pantalla de acceso.",
  privacyExportTitle: "Exportar mis datos",
  privacyExportBody:
    "Un archivo JSON con tu perfil, tus membresías, la lista de tus sesiones y tokens (nunca sus valores) y todo lo de los espacios de trabajo que posees. Nunca se incluyen contraseñas, secretos de dos factores, tokens ni claves de IA.",
  privacyExportButton: "Exportar mis datos",
  privacyExportStarted: "La descarga comenzó. Si no, confirma tu contraseña de nuevo.",
  privacyWorkspaceTitle: "Eliminar este espacio de trabajo",
  privacyWorkspaceBody:
    "Solo el propietario puede hacerlo. El espacio pasa a solo lectura para todos ahora, sus miembros reciben un correo y todo lo que contiene se borra para siempre después de {days} días. El propietario puede cancelar hasta entonces. La suscripción se cancela de inmediato; no hay reembolso automático.",
  privacyWorkspaceName: "Escribe el nombre del espacio de trabajo para confirmar",
  privacyWorkspaceDelete: "Eliminar el espacio de trabajo",
  privacyWorkspaceOwnerOnly: "Solo el propietario de este espacio de trabajo puede eliminarlo.",
  privacyWorkspacePersonal:
    "La edición personal tiene un espacio de trabajo. Para quitarlo con todo lo que contiene, borra todos los datos locales abajo.",
  privacyPendingTitle: "Este espacio de trabajo está programado para eliminarse",
  privacyPendingBody:
    "Está en solo lectura y se borrará para siempre el {date}. Hasta entonces aún puedes leer y exportar tus datos.",
  privacyPendingCancel: "Cancelar la eliminación",
  privacyPendingOwnerOnly: "Solo el propietario puede cancelar la eliminación.",
  privacyAccountTitle: "Eliminar mi cuenta",
  privacyAccountBody:
    "Todas las sesiones y tokens terminan ahora y la cuenta se borra para siempre después de {days} días; entra antes para cancelar. Los espacios donde eres el único miembro se van con ella. Si eres el único propietario de un espacio que usan otras personas, transfiere la propiedad o elimina ese espacio antes. En los espacios que quedan, tus datos privados (conversaciones del asistente, notificaciones, tokens, dispositivos) se borran y tu nombre sale de lo que escribiste.",
  privacyAccountConfirm: "Escribe tu correo para confirmar",
  privacyAccountDelete: "Eliminar mi cuenta",
  accountDeletionTitle: "Tu cuenta está programada para eliminarse",
  accountDeletionBody:
    "Se borrará para siempre el {date}. Entrar no la cancela. Tus sesiones, tokens y apps conectadas terminaron y no se restauran.",
  accountDeletionCancel: "Conservar mi cuenta",
  accountDeletionSignOut: "Salir",
  privacyEraseTitle: "Borrar todos los datos locales",
  privacyEraseBody:
    "Quita de este dispositivo la base de datos de esta instalación, con todo lo que contiene. No se envía nada a ningún sitio y no hace falta red. La app se reinicia vacía, en la configuración inicial. Exporta tus datos antes si quieres conservarlos.",
  privacyEraseConfirm: "Escribe ERASE para confirmar",
  privacyEraseButton: "Borrar todos los datos locales",
  privacyErased: "Los datos locales se borraron. La app se reinicia vacía: recarga esta página en unos segundos.",
} satisfies Record<keyof typeof en, string>;
export default dict;

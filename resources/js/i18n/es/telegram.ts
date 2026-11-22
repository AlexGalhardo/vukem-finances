import type en from "../en/telegram";

const dict = {
  telegram: "Telegram",
  telegramIntro:
    "Registra un gasto o un ingreso con un mensaje corto, consulta tus saldos y deshaz un error, sin abrir la app.",
  telegramPrivacy:
    "Los mensajes pasan por los servidores de Telegram y no tienen cifrado de extremo a extremo, y el comando /balance muestra tus saldos en el chat. Puedes desconectar cuando quieras y borrar el historial del chat en Telegram.",
  telegramConnect: "Conectar un chat",
  telegramConnectHint: "Recibes un enlace que sirve una vez, durante 10 minutos. Ábrelo en Telegram y pulsa Iniciar.",
  telegramCreatedTitle: "Abre este enlace en Telegram",
  telegramCreatedBody:
    "Se muestra solo ahora y sirve una vez. No lo compartas: quien lo abra primero conecta su propio chat.",
  telegramOpen: "Abrir en Telegram",
  telegramCopy: "Copiar enlace",
  telegramCodeHint: "O envía esto al bot:",
  telegramExpiresIn: "Caduca en {time}",
  telegramExpired: "Este código caducó. Conecta de nuevo para obtener otro.",
  telegramDismiss: "Listo",
  telegramLinksTitle: "Chats conectados",
  telegramNone: "No hay ningún chat conectado.",
  telegramChat: "Chat terminado en {hint}",
  telegramLinkedOn: "Conectado el",
  telegramLastSeen: "Último mensaje",
  telegramNeverSeen: "aún sin mensajes",
  telegramDisconnect: "Desconectar",
  telegramLimit: "Puedes conectar como máximo {max} chats.",
  telegramAlertsTitle: "Alertas",
  telegramAlertsLabel: "Enviar las alertas de este workspace a mis chats conectados",
  telegramAlertsHint: "Alertas de presupuesto, tarjeta, meta y recurrencias. Desactivado hasta que lo actives.",
  telegramSettings: "Ajustes de Telegram",
} satisfies Record<keyof typeof en, string>;
export default dict;

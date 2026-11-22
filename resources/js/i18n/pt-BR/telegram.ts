import type en from "../en/telegram";

const dict = {
  telegram: "Telegram",
  telegramIntro:
    "Registre uma despesa ou uma receita com uma mensagem curta, consulte seus saldos e desfaça um erro, sem abrir o app.",
  telegramPrivacy:
    "As mensagens passam pelos servidores do Telegram e não têm criptografia de ponta a ponta, e o comando /balance mostra seus saldos no chat. Você pode desconectar quando quiser e apagar o histórico da conversa no Telegram.",
  telegramConnect: "Conectar um chat",
  telegramConnectHint: "Você recebe um link que vale uma vez, por 10 minutos. Abra no Telegram e toque em Iniciar.",
  telegramCreatedTitle: "Abra este link no Telegram",
  telegramCreatedBody:
    "Ele aparece só agora e vale uma vez. Não compartilhe: quem abrir primeiro conecta o próprio chat.",
  telegramOpen: "Abrir no Telegram",
  telegramCopy: "Copiar link",
  telegramCodeHint: "Ou envie isto ao bot:",
  telegramExpiresIn: "Expira em {time}",
  telegramExpired: "Este código expirou. Conecte de novo para gerar outro.",
  telegramDismiss: "Pronto",
  telegramLinksTitle: "Chats conectados",
  telegramNone: "Nenhum chat conectado.",
  telegramChat: "Chat final {hint}",
  telegramLinkedOn: "Conectado em",
  telegramLastSeen: "Última mensagem",
  telegramNeverSeen: "nenhuma mensagem ainda",
  telegramDisconnect: "Desconectar",
  telegramLimit: "Você pode conectar no máximo {max} chats.",
  telegramAlertsTitle: "Alertas",
  telegramAlertsLabel: "Enviar os alertas deste workspace para os meus chats conectados",
  telegramAlertsHint: "Alertas de orçamento, cartão, meta e recorrência. Desligado até você ativar.",
  telegramSettings: "Configurações do Telegram",
} satisfies Record<keyof typeof en, string>;
export default dict;

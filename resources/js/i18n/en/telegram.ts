const dict = {
  telegram: "Telegram",
  telegramIntro:
    "Record an expense or an income with one short message, check your balances and undo a mistake, without opening the app.",
  telegramPrivacy:
    "Messages pass through Telegram's servers and are not end-to-end encrypted, and the /balance command shows your balances in the chat. You can disconnect at any time and delete the chat history in Telegram.",
  telegramConnect: "Connect a chat",
  telegramConnectHint: "You get a link that works once, for 10 minutes. Open it in Telegram and press Start.",
  telegramCreatedTitle: "Open this link in Telegram",
  telegramCreatedBody: "It is shown only now and works once. Do not share it: whoever opens it first links their chat.",
  telegramOpen: "Open in Telegram",
  telegramCopy: "Copy link",
  telegramCodeHint: "Or send this to the bot yourself:",
  telegramExpiresIn: "Expires in {time}",
  telegramExpired: "This code expired. Connect again to get a new one.",
  telegramDismiss: "Done",
  telegramLinksTitle: "Linked chats",
  telegramNone: "No chat is linked.",
  telegramChat: "Chat ending in {hint}",
  telegramLinkedOn: "Linked",
  telegramLastSeen: "Last message",
  telegramNeverSeen: "no message yet",
  telegramDisconnect: "Disconnect",
  telegramLimit: "You can link at most {max} chats.",
  telegramAlertsTitle: "Alerts",
  telegramAlertsLabel: "Send my alerts of this workspace to my linked chats",
  telegramAlertsHint: "Budget, card, goal and recurring alerts. Off until you turn it on.",
  telegramSettings: "Telegram settings",
  telegramNeedsInternet:
    "This installation works offline. Telegram is the exception: the bot talks to Telegram's servers, so it needs the internet. Without it the bot waits and nothing else is affected.",
} as const;
export default dict;

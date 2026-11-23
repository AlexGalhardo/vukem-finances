// A fake Telegram Bot API for the Telegram settings spec: getMe, getUpdates (a short long poll), sendMessage and the
// calls that only need an ok. It listens on loopback only and never reaches the network; the token is a placeholder.
// Control: POST /__control/push {"chat": 424242, "text": "/start CODE"} queues a message from that chat;
// POST /__control/tap {"chat": 424242, "data": "..."} queues a tap on an inline button.
// GET /__control/sent lists what the bot sent, with the inline buttons of each message.
import { createServer } from "node:http";

export const FAKE_TELEGRAM_TOKEN = "123456:e2e-placeholder-token";

export async function startFakeTelegram(port) {
  const queue = [];
  const sent = [];
  let seq = 1000;

  const reply = (res, status, body) => {
    res.statusCode = status;
    res.setHeader("Content-Type", "application/json");
    res.end(JSON.stringify(body));
  };
  const ok = (res, result) => reply(res, 200, { ok: true, result });

  const server = createServer(async (req, res) => {
    const url = new URL(req.url ?? "/", "http://127.0.0.1");
    const chunks = [];
    for await (const chunk of req) chunks.push(chunk);
    const body = chunks.length ? JSON.parse(Buffer.concat(chunks).toString("utf8")) : {};

    if (url.pathname === "/__control/push") {
      const id = ++seq;
      queue.push({
        update_id: id,
        message: {
          message_id: id,
          date: Math.floor(Date.now() / 1000),
          from: { id: body.chat, is_bot: false, first_name: "E2E" },
          chat: { id: body.chat, type: "private" },
          text: body.text,
        },
      });
      return reply(res, 200, { ok: true, update_id: id });
    }
    if (url.pathname === "/__control/tap") {
      const id = ++seq;
      queue.push({
        update_id: id,
        callback_query: {
          id: String(id),
          from: { id: body.chat, is_bot: false, first_name: "E2E" },
          message: { message_id: id, date: Math.floor(Date.now() / 1000), chat: { id: body.chat, type: "private" } },
          data: body.data,
        },
      });
      return reply(res, 200, { ok: true, update_id: id });
    }
    if (url.pathname === "/__control/sent") return reply(res, 200, sent);

    const prefix = `/bot${FAKE_TELEGRAM_TOKEN}/`;
    if (!url.pathname.startsWith(prefix)) {
      return reply(res, 401, { ok: false, error_code: 401, description: "Unauthorized" });
    }
    switch (url.pathname.slice(prefix.length)) {
      case "getMe":
        return ok(res, { id: 123456, is_bot: true, first_name: "E2E", username: "e2e_finance_bot" });
      case "getUpdates": {
        const offset = body.offset ?? 0;
        for (let i = 0; i < 10; i++) {
          const ready = queue.filter((u) => u.update_id >= offset);
          if (ready.length) return ok(res, ready);
          await new Promise((resolve) => setTimeout(resolve, 100));
        }
        return ok(res, []);
      }
      case "sendMessage":
        sent.push({
          chat: body.chat_id,
          text: body.text,
          buttons: (body.reply_markup?.inline_keyboard ?? [])
            .flat()
            .map((b) => ({ text: b.text, data: b.callback_data })),
        });
        return ok(res, { message_id: ++seq, chat: { id: body.chat_id }, text: body.text });
      default:
        return ok(res, true);
    }
  });

  await new Promise((resolve) => server.listen(port, "127.0.0.1", resolve));
  return server;
}

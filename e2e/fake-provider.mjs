// A fake OpenAI-compatible server for the AI spec: no real provider is ever called.
import { createServer } from "node:http";

/** The system and user text of an OpenAI-style chat request. */
function parse(raw) {
  try {
    const messages = JSON.parse(raw).messages ?? [];
    const text = (role) =>
      messages
        .filter((m) => m.role === role)
        .map((m) => m.content)
        .join(" ");
    return { system: text("system"), user: text("user") };
  } catch {
    return { system: "", user: "" };
  }
}

/** Starts the server on a free loopback port. `setStatus(401)` makes it answer like a provider that rejects the key. */
export async function startFakeProvider(echo) {
  let status = 200;
  let reply = null;
  const server = createServer((req, res) => {
    const chunks = [];
    req.on("data", (c) => chunks.push(c));
    req.on("end", () => respond(Buffer.concat(chunks).toString("utf8"), res));
  });
  const respond = (raw, res) => {
    res.setHeader("Content-Type", "application/json");
    res.statusCode = status;
    res.end(
      status === 200
        ? JSON.stringify({
            model: "fake",
            choices: [{ message: { content: reply ? reply(parse(raw)) : "pong" } }],
            usage: { prompt_tokens: 1, completion_tokens: 1 },
          })
        : JSON.stringify({ error: { type: "authentication_error", message: `bad key ${echo}` } }),
    );
  };
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  return {
    baseUrl: `http://127.0.0.1:${server.address().port}/v1`,
    setStatus: (value) => {
      status = value;
    },
    /** `fn({ system, user })` returns the text of the next answers; null goes back to "pong". */
    setReply: (fn) => {
      reply = fn;
    },
    close: () => new Promise((resolve) => server.close(() => resolve())),
  };
}

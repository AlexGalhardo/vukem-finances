// A fake AbacatePay for the billing spec: create, check and simulate-payment like the real API, plus a control
// endpoint to set the state of a charge. It listens on loopback only and never reaches the network.
import { createServer } from "node:http";

const PNG =
  "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP4z8DwHwAFAAH/q842iQAAAABJRU5ErkJggg==";

/** Starts the fake on `port`. Control: GET /__control/charges lists the charges; POST /__control/status?id=&status= sets one. */
export async function startFakeAbacatePay(port) {
  const charges = new Map();
  let seq = 0;

  const reply = (res, status, body) => {
    res.statusCode = status;
    res.setHeader("Content-Type", "application/json");
    res.end(JSON.stringify(body));
  };
  const view = (c) => ({
    id: c.id,
    amount: c.amount,
    status: c.status,
    brCode: `00020101021226fake${c.id}`,
    brCodeBase64: PNG,
    expiresAt: c.expiresAt,
  });

  const server = createServer(async (req, res) => {
    const url = new URL(req.url ?? "/", "http://127.0.0.1");
    const chunks = [];
    for await (const chunk of req) chunks.push(chunk);
    const body = chunks.length ? JSON.parse(Buffer.concat(chunks).toString("utf8")) : {};

    if (url.pathname.startsWith("/__control/")) {
      if (url.pathname.endsWith("/charges")) return reply(res, 200, [...charges.values()]);
      const c = charges.get(url.searchParams.get("id") ?? "");
      if (c) c.status = url.searchParams.get("status") ?? c.status;
      return reply(res, c ? 200 : 404, { ok: Boolean(c) });
    }

    if (req.headers.authorization !== "Bearer abc_dev_e2e_placeholder_key") return reply(res, 401, { error: "no" });

    if (req.method === "POST" && url.pathname === "/transparents/create") {
      const data = body.data ?? {};
      const c = {
        id: `pix_char_${++seq}`,
        amount: data.amount,
        status: "PENDING",
        expiresAt: new Date(Date.now() + (data.expiresIn ?? 1800) * 1000).toISOString(),
      };
      charges.set(c.id, c);
      return reply(res, 200, { success: true, error: null, data: view(c) });
    }
    const c = charges.get(url.searchParams.get("id") ?? "");
    if (!c) return reply(res, 404, { error: "Pix QR Code not found" });
    if (req.method === "GET" && url.pathname === "/transparents/check") {
      return reply(res, 200, { success: true, error: null, data: view(c) });
    }
    if (req.method === "POST" && url.pathname === "/transparents/simulate-payment") {
      if (c.status === "PENDING") c.status = "PAID";
      return reply(res, 200, { success: true, error: null, data: view(c) });
    }
    return reply(res, 404, { error: "not found" });
  });

  await new Promise((resolve) => server.listen(port, "127.0.0.1", resolve));
  return { port, close: () => server.close() };
}

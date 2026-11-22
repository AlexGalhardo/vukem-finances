import { checkServerUrl, HttpTransport, type TokenStore } from "../../src/sync/transport";
import { TransportError } from "../../src/sync/types";

const SECRET = ["tok", "en", "-", "value", "-", "1234"].join("");
const CODE = ["ABCD", "EFGH"].join("-");

function store(value: string | null = SECRET): TokenStore {
  let token = value;
  return {
    get: async () => token,
    set: async (t) => {
      token = t;
    },
    clear: async () => {
      token = null;
    },
  };
}

interface Seen {
  url: string;
  init: RequestInit;
}

function fakeFetch(answer: (seen: Seen) => Response | Promise<Response>): { fetchImpl: typeof fetch; seen: Seen[] } {
  const seen: Seen[] = [];
  const fetchImpl = (async (url: string, init: RequestInit) => {
    const s = { url, init };
    seen.push(s);
    return answer(s);
  }) as unknown as typeof fetch;
  return { fetchImpl, seen };
}

const json = (body: unknown, status = 200, headers: Record<string, string> = {}): Response =>
  new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json", ...headers } });

describe("server address rules", () => {
  it.each([
    ["https://money.example.com", true, true],
    ["https://money.example.com/base/", true, true],
    ["http://localhost:8080", true, false],
    ["http://192.168.1.20:3000", true, false],
    ["http://10.0.0.5", true, false],
    ["http://172.16.4.1", true, false],
    ["http://nas.local", true, false],
    ["http://[::1]:3000", true, false],
  ])("%s is allowed (secure %s)", (raw, ok, secure) => {
    const r = checkServerUrl(raw);
    expect(r.ok).toBe(ok);
    expect(r.ok && r.secure).toBe(secure);
  });

  it.each([
    ["http://money.example.com", "public_http"],
    ["http://172.32.0.1", "public_http"],
    ["http://8.8.8.8", "public_http"],
    ["ftp://nas.local", "scheme"],
    ["money.example.com", "scheme"],
    ["", "empty"],
    ["https://", "malformed"],
    ["https://user@host.example.com", "malformed"],
  ])("%s is refused as %s", (raw, reason) => {
    expect(checkServerUrl(raw)).toEqual({ ok: false, reason });
  });

  it("drops the trailing slash and keeps a path prefix", () => {
    expect(checkServerUrl("https://money.example.com/base///")).toEqual({
      ok: true,
      url: "https://money.example.com/base",
      secure: true,
    });
  });
});

describe("HttpTransport", () => {
  it("refuses http to a public host and http to a private one without the confirmation", () => {
    const base = { tokens: store(), deviceId: () => null };
    expect(() => new HttpTransport({ ...base, baseUrl: "http://money.example.com", allowInsecure: true })).toThrow();
    expect(() => new HttpTransport({ ...base, baseUrl: "http://192.168.1.20" })).toThrow(/confirmation/u);
    expect(() => new HttpTransport({ ...base, baseUrl: "http://192.168.1.20", allowInsecure: true })).not.toThrow();
    expect(() => new HttpTransport({ ...base, baseUrl: "https://money.example.com" })).not.toThrow();
  });

  it("sends the protocol headers, the bearer token and the device", async () => {
    const { fetchImpl, seen } = fakeFetch(() =>
      json({ changes: [], next_cursor: 0, has_more: false, server_time: "" }),
    );
    const t = new HttpTransport({
      baseUrl: "https://money.example.com/base",
      tokens: store(),
      deviceId: () => "DEVICE1",
      fetchImpl,
    });
    await t.pull(41, 500);

    expect(seen[0]?.url).toBe("https://money.example.com/base/api/v1/sync/pull?cursor=41&limit=500");
    expect(seen[0]?.init.headers).toMatchObject({
      Authorization: `Bearer ${SECRET}`,
      "X-Sync-Protocol": "1",
      "X-Device-Id": "DEVICE1",
    });
  });

  it("sends no device header before the device exists, and pair sends no token at all", async () => {
    const { fetchImpl, seen } = fakeFetch(() => json({ token: "t", device_id: "d", workspace: {}, server_time: "" }));
    const t = new HttpTransport({
      baseUrl: "https://money.example.com",
      tokens: store(),
      deviceId: () => "D",
      fetchImpl,
    });
    await t.registerDevice("Phone", "android");
    await t.pair(CODE, "Phone", "android");

    expect(seen[0]?.init.headers).not.toHaveProperty("X-Device-Id");
    expect(seen[0]?.init.headers).toHaveProperty("Authorization");
    expect(seen[1]?.url).toBe("https://money.example.com/api/v1/sync/pair");
    expect(seen[1]?.init.headers).not.toHaveProperty("Authorization");
    expect(seen[1]?.init.headers).not.toHaveProperty("X-Device-Id");
    expect(JSON.parse(String(seen[1]?.init.body))).toEqual({ code: CODE, device_name: "Phone", platform: "android" });
  });

  it("reads a problem into a typed error and never puts the token or the code in it", async () => {
    const { fetchImpl } = fakeFetch(() =>
      json({ code: "rate_limited", detail: `slow down ${SECRET} ${CODE}` }, 429, { "Retry-After": "7" }),
    );
    const t = new HttpTransport({
      baseUrl: "https://money.example.com",
      tokens: store(),
      deviceId: () => "D",
      fetchImpl,
    });
    const error = (await t.pair(CODE, "Phone", "ios").catch((e: unknown) => e)) as TransportError;

    expect(error).toBeInstanceOf(TransportError);
    expect([error.kind, error.status, error.code, error.retryAfterSeconds]).toEqual(["http", 429, "rate_limited", 7]);
    expect(error.message).not.toContain(SECRET);
    expect(error.message).not.toContain(CODE);
  });

  it("redacts the token from a network failure message", async () => {
    const { fetchImpl } = fakeFetch(() => {
      throw new Error(`connection refused for Bearer ${SECRET}`);
    });
    const t = new HttpTransport({
      baseUrl: "https://money.example.com",
      tokens: store(),
      deviceId: () => "D",
      fetchImpl,
    });
    const error = (await t.pull(0, 10).catch((e: unknown) => e)) as TransportError;

    expect(error.kind).toBe("network");
    expect(error.message).not.toContain(SECRET);
  });

  it("times out a request that never answers", async () => {
    const fetchImpl = ((_url: string, init: RequestInit) =>
      new Promise((_resolve, reject) => {
        init.signal?.addEventListener("abort", () => reject(new Error("aborted")));
      })) as unknown as typeof fetch;
    const t = new HttpTransport({
      baseUrl: "https://money.example.com",
      tokens: store(),
      deviceId: () => "D",
      fetchImpl,
      timeoutMs: 20,
    });
    const error = (await t.pull(0, 10).catch((e: unknown) => e)) as TransportError;

    expect(error.kind).toBe("timeout");
  });

  it("returns the results of a push", async () => {
    const { fetchImpl, seen } = fakeFetch(() => json({ results: [{ op_id: "o1", status: "applied" }] }));
    const t = new HttpTransport({
      baseUrl: "https://money.example.com",
      tokens: store(),
      deviceId: () => "D",
      fetchImpl,
    });
    const out = await t.push([{ op_id: "o1", entity: "account", uid: "u", op: "upsert", base_seq: 0, fields: {} }]);

    expect(out[0]?.status).toBe("applied");
    expect(seen[0]?.init.method).toBe("POST");
  });
});

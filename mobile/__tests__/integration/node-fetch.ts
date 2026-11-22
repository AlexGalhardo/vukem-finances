import { request } from "node:http";

interface Init {
  method?: string;
  headers?: Record<string, string>;
  body?: string;
  signal?: AbortSignal;
}

export class NodeResponse {
  readonly ok: boolean;
  readonly headers: { get(name: string): string | null; getSetCookie(): string[] };

  constructor(
    readonly status: number,
    private readonly raw: Record<string, string | string[] | undefined>,
    private readonly text_: string,
  ) {
    this.ok = status >= 200 && status < 300;
    this.headers = {
      get: (name) => {
        const v = raw[name.toLowerCase()];
        return Array.isArray(v) ? (v[0] ?? null) : (v ?? null);
      },
      getSetCookie: () => {
        const v = raw["set-cookie"];
        return Array.isArray(v) ? v : v === undefined ? [] : [v];
      },
    };
  }

  async text(): Promise<string> {
    return this.text_;
  }

  async json(): Promise<unknown> {
    return JSON.parse(this.text_);
  }
}

/**
 * A fetch over Node's http module. The Jest environment of React Native replaces
 * the global fetch with an XMLHttpRequest polyfill that cannot read Set-Cookie or
 * stop at a redirect, and this suite needs both to sign in like a browser does.
 */
export async function nodeFetch(url: string, init: Init = {}): Promise<NodeResponse> {
  return new Promise((resolve, reject) => {
    const req = request(url, { method: init.method ?? "GET", headers: init.headers ?? {} }, (res) => {
      const chunks: Uint8Array[] = [];
      res.on("data", (c) => chunks.push(c));
      res.on("end", () => {
        const size = chunks.reduce((n, c) => n + c.length, 0);
        const all = new Uint8Array(size);
        let at = 0;
        for (const c of chunks) {
          all.set(c, at);
          at += c.length;
        }
        resolve(new NodeResponse(res.statusCode ?? 0, res.headers, new TextDecoder().decode(all)));
      });
      res.on("error", reject);
    });
    req.on("error", reject);
    init.signal?.addEventListener("abort", () => req.destroy(new Error("aborted")));
    if (init.body !== undefined) req.write(init.body);
    req.end();
  });
}

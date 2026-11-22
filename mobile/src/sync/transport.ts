import {
  type DevicePlatform,
  type Mutation,
  type PairResult,
  type PullPage,
  type PushResult,
  type ServerInfo,
  type SnapshotPage,
  type SnapshotQuery,
  type Transport,
  TransportError,
} from "./types";

export const PROTOCOL_VERSION = 1;
export const REQUEST_TIMEOUT_MS = 30_000;

export interface TokenStore {
  get(): Promise<string | null>;
  set(token: string): Promise<void>;
  clear(): Promise<void>;
}

export type ServerUrlProblem = "empty" | "malformed" | "scheme" | "public_http";

export type ServerUrlCheck = { ok: true; url: string; secure: boolean } | { ok: false; reason: ServerUrlProblem };

const PATTERN = /^([a-z][a-z0-9+.-]*):\/\/([^/?#@\\\s]+)(\/[^?#\s]*)?$/iu;

function splitHost(authority: string): string | null {
  if (authority.startsWith("[")) {
    const end = authority.indexOf("]");
    return end > 1 ? authority.slice(1, end).toLowerCase() : null;
  }
  const host = authority.split(":")[0] ?? "";
  return host === "" ? null : host.toLowerCase();
}

/** localhost, loopback, link-local, RFC 1918 and unique-local addresses and `.local` names. */
export function isPrivateHost(host: string): boolean {
  if (host === "localhost" || host.endsWith(".localhost") || host.endsWith(".local")) {
    return true;
  }
  const v4 = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/u.exec(host);
  if (v4 !== null) {
    const a = Number(v4[1]);
    const b = Number(v4[2]);
    return (
      a === 10 || a === 127 || (a === 172 && b >= 16 && b <= 31) || (a === 192 && b === 168) || (a === 169 && b === 254)
    );
  }
  if (host.includes(":")) {
    return host === "::1" || /^f[cd][0-9a-f]{2}:/u.test(host) || /^fe[89ab][0-9a-f]:/u.test(host);
  }
  return false;
}

/**
 * The server address rules: https always; http only for a private address
 * (the caller must then show the plain-traffic confirmation: `secure` is
 * false); http to a public host is refused.
 */
export function checkServerUrl(raw: string): ServerUrlCheck {
  const text = raw.trim();
  if (text === "") {
    return { ok: false, reason: "empty" };
  }
  const match = PATTERN.exec(text);
  if (match === null) {
    return { ok: false, reason: /^[a-z][a-z0-9+.-]*:\/\//iu.test(text) ? "malformed" : "scheme" };
  }
  const scheme = (match[1] ?? "").toLowerCase();
  const host = splitHost(match[2] ?? "");
  if (host === null) {
    return { ok: false, reason: "malformed" };
  }
  const path = (match[3] ?? "").replace(/\/+$/u, "");
  const url = `${scheme}://${match[2]}${path}`;
  if (scheme === "https") {
    return { ok: true, url, secure: true };
  }
  if (scheme !== "http") {
    return { ok: false, reason: "scheme" };
  }
  return isPrivateHost(host) ? { ok: true, url, secure: false } : { ok: false, reason: "public_http" };
}

export interface HttpTransportOptions {
  baseUrl: string;
  /** Allows an http address the person confirmed; an https address needs no flag. */
  allowInsecure?: boolean;
  tokens: TokenStore;
  deviceId: () => string | null;
  fetchImpl?: typeof fetch;
  timeoutMs?: number;
}

function redact(text: string, secrets: readonly (string | null)[]): string {
  let out = text;
  for (const s of secrets) {
    if (s !== null && s.length >= 4) {
      out = out.split(s).join("[redacted]");
    }
  }
  return out;
}

/** `fetch` over JSON with the protocol headers. */
export class HttpTransport implements Transport {
  private readonly root: string;
  private readonly fetchImpl: typeof fetch;
  private readonly timeoutMs: number;

  constructor(private readonly options: HttpTransportOptions) {
    const checked = checkServerUrl(options.baseUrl);
    if (!checked.ok) {
      throw new Error(`the server address is not allowed: ${checked.reason}`);
    }
    if (!checked.secure && options.allowInsecure !== true) {
      throw new Error("the server address needs the plain-traffic confirmation");
    }
    this.root = `${checked.url}/api/v1/sync`;
    this.fetchImpl = options.fetchImpl ?? fetch;
    this.timeoutMs = options.timeoutMs ?? REQUEST_TIMEOUT_MS;
  }

  info(): Promise<ServerInfo> {
    return this.call<ServerInfo>("GET", "/info", undefined, { device: false });
  }

  registerDevice(name: string, platform: DevicePlatform): Promise<{ device_id: string }> {
    return this.call("POST", "/devices", { name, platform }, { device: false });
  }

  pull(cursor: number, limit: number): Promise<PullPage> {
    return this.call("GET", `/pull?cursor=${cursor}&limit=${limit}`);
  }

  snapshot(q: SnapshotQuery): Promise<SnapshotPage> {
    const params = [`entity=${encodeURIComponent(q.entity)}`, `limit=${q.limit}`];
    if (q.after !== "") params.push(`after=${encodeURIComponent(q.after)}`);
    if (q.snapshot_id !== "") params.push(`snapshot_id=${encodeURIComponent(q.snapshot_id)}`);
    return this.call("GET", `/snapshot?${params.join("&")}`);
  }

  async push(mutations: readonly Mutation[]): Promise<PushResult[]> {
    const out = await this.call<{ results: PushResult[] }>("POST", "/push", { mutations });
    return out.results;
  }

  pair(code: string, deviceName: string, platform: DevicePlatform): Promise<PairResult> {
    return this.call(
      "POST",
      "/pair",
      { code, device_name: deviceName, platform },
      { auth: false, device: false, secret: code },
    );
  }

  private async call<T>(
    method: "GET" | "POST",
    path: string,
    body?: unknown,
    flags: { auth?: boolean; device?: boolean; secret?: string } = {},
  ): Promise<T> {
    const stored = await this.options.tokens.get();
    const token = flags.auth === false ? null : stored;
    const device = flags.device === false ? null : this.options.deviceId();
    const secrets = [stored, flags.secret ?? null];

    const headers: Record<string, string> = {
      Accept: "application/json",
      "X-Sync-Protocol": String(PROTOCOL_VERSION),
    };
    if (body !== undefined) headers["Content-Type"] = "application/json";
    if (token !== null) headers.Authorization = `Bearer ${token}`;
    if (device !== null) headers["X-Device-Id"] = device;

    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeoutMs);
    let response: Response;
    try {
      response = await this.fetchImpl(`${this.root}${path}`, {
        method,
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
        signal: controller.signal,
      });
    } catch (error) {
      if (controller.signal.aborted) {
        throw new TransportError("timeout", "the server did not answer in time");
      }
      throw new TransportError("network", redact(error instanceof Error ? error.message : "network error", secrets));
    } finally {
      clearTimeout(timer);
    }

    let parsed: unknown = null;
    try {
      parsed = await response.json();
    } catch {
      parsed = null;
    }
    if (!response.ok) {
      const problem = (parsed ?? {}) as { code?: unknown; detail?: unknown };
      throw new TransportError(
        "http",
        redact(typeof problem.detail === "string" ? problem.detail : `HTTP ${response.status}`, secrets),
        response.status,
        typeof problem.code === "string" ? problem.code : "",
        Number(response.headers.get("Retry-After") ?? 0) || 0,
      );
    }
    if (parsed === null) {
      throw new TransportError("network", "the server answered with something that is not JSON");
    }
    return parsed as T;
  }
}

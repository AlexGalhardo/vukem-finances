import type {
  DevicePlatform,
  Mutation,
  PairResult,
  PullPage,
  PushResult,
  ServerInfo,
  SnapshotPage,
  SnapshotQuery,
  Transport,
} from "../../src/sync/types";

interface Handlers {
  info: () => ServerInfo | Promise<ServerInfo>;
  registerDevice: (name: string, platform: DevicePlatform) => { device_id: string } | Promise<{ device_id: string }>;
  pull: (cursor: number, limit: number) => PullPage | Promise<PullPage>;
  snapshot: (q: SnapshotQuery) => SnapshotPage | Promise<SnapshotPage>;
  push: (mutations: readonly Mutation[]) => PushResult[] | Promise<PushResult[]>;
  pair: (code: string, name: string, platform: DevicePlatform) => PairResult | Promise<PairResult>;
}

/** A transport whose every answer a test scripts; it records the calls. */
export class FakeTransport implements Transport {
  readonly calls: string[] = [];

  constructor(readonly handlers: Partial<Handlers> = {}) {}

  private async run<K extends keyof Handlers>(
    name: K,
    ...args: Parameters<Handlers[K]>
  ): Promise<Awaited<ReturnType<Handlers[K]>>> {
    this.calls.push(name);
    const handler = this.handlers[name] as ((...a: unknown[]) => unknown) | undefined;
    if (handler === undefined) {
      throw new Error(`FakeTransport: ${name} is not scripted`);
    }
    return (await handler(...args)) as Awaited<ReturnType<Handlers[K]>>;
  }

  info(): Promise<ServerInfo> {
    return this.run("info");
  }

  registerDevice(name: string, platform: DevicePlatform): Promise<{ device_id: string }> {
    return this.run("registerDevice", name, platform);
  }

  pull(cursor: number, limit: number): Promise<PullPage> {
    return this.run("pull", cursor, limit);
  }

  snapshot(q: SnapshotQuery): Promise<SnapshotPage> {
    return this.run("snapshot", q);
  }

  push(mutations: readonly Mutation[]): Promise<PushResult[]> {
    return this.run("push", mutations);
  }

  pair(code: string, name: string, platform: DevicePlatform): Promise<PairResult> {
    return this.run("pair", code, name, platform);
  }
}

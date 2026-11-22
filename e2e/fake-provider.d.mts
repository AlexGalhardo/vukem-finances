export interface FakeProvider {
  baseUrl: string;
  setStatus(status: number): void;
  /** Answers with the text `fn` returns for each request; null goes back to the default. */
  setReply(fn: ((request: { system: string; user: string }) => string) | null): void;
  close(): Promise<void>;
}

export function startFakeProvider(echo: string): Promise<FakeProvider>;

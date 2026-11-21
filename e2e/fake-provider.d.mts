export interface FakeProvider {
  baseUrl: string;
  setStatus(status: number): void;
  close(): Promise<void>;
}

export function startFakeProvider(echo: string): Promise<FakeProvider>;

// The integration suite talks to the real server through Node's http module; @types/node is not a dependency.
declare module "node:http" {
  interface IncomingMessage {
    statusCode?: number;
    headers: Record<string, string | string[] | undefined>;
    rawHeaders: string[];
    on(event: "data", listener: (chunk: Uint8Array) => void): void;
    on(event: "end", listener: () => void): void;
    on(event: "error", listener: (error: Error) => void): void;
  }
  interface ClientRequest {
    on(event: "error", listener: (error: Error) => void): void;
    write(chunk: string): void;
    end(): void;
    destroy(error?: Error): void;
  }
  export function request(
    url: string,
    options: { method?: string; headers?: Record<string, string> },
    callback: (response: IncomingMessage) => void,
  ): ClientRequest;
}

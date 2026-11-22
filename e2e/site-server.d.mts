export interface Served {
  origin: string;
  close(): Promise<void>;
}

export const OUT: string;
export function serve(root: string): Promise<Served>;
export function ensureBuilt(): void;
export function builtPages(): string[];
export function servePlanted(html: string): Promise<Served>;

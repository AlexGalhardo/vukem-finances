import type ai from "./en/ai";
import type common from "./en/common";
import type components from "./en/components";
import type shell from "./en/shell";

/** Every translation key. `en` is the typed source of truth; add one `typeof` line per new domain file. */
export type Key = keyof typeof ai | keyof typeof common | keyof typeof components | keyof typeof shell;

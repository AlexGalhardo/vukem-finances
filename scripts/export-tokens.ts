import { readFileSync, writeFileSync } from "node:fs";
import { buildTokens } from "./tokens-lib";

const css = readFileSync(new URL("../resources/css/tokens.css", import.meta.url), "utf8");
writeFileSync(new URL("../mobile/src/theme/tokens.ts", import.meta.url), buildTokens(css));

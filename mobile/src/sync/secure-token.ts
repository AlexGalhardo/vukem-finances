import { deleteItemAsync, getItemAsync, setItemAsync } from "expo-secure-store";
import type { TokenStore } from "./transport";

const KEY = "vukem-finances.sync.token";

/**
 * The token lives only here: never in SQLite, never in a backup, never in a log.
 * It needs a device, so it is on the manual checklist.
 */
export const secureTokenStore: TokenStore = {
  get: () => getItemAsync(KEY),
  set: (token) => setItemAsync(KEY, token),
  clear: () => deleteItemAsync(KEY),
};

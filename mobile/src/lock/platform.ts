import { authenticateAsync, getEnrolledLevelAsync, SecurityLevel } from "expo-local-authentication";
import { deleteItemAsync, getItemAsync, setItemAsync } from "expo-secure-store";
import type { LockPlatform, LockStore } from "./lock";

const KEY = "vukem-finances.lock.enabled";

/** The device implementation: needs hardware and a real device, so it is covered by the manual checklist. */
export const devicePlatform: LockPlatform = {
	async available() {
		return (await getEnrolledLevelAsync()) !== SecurityLevel.NONE;
	},
	async authenticate(prompt) {
		const result = await authenticateAsync({ promptMessage: prompt, disableDeviceFallback: false });
		return result.success;
	},
};

export const secureLockStore: LockStore = {
	async isEnabled() {
		return (await getItemAsync(KEY)) === "1";
	},
	async setEnabled(enabled) {
		if (enabled) {
			await setItemAsync(KEY, "1");
		} else {
			await deleteItemAsync(KEY);
		}
	},
};

/** After this long in the background the app locks again (spec: 60 seconds). */
export const LOCK_AFTER_MS = 60_000;

export interface LockPlatform {
  /** Whether the device has a screen lock or biometrics the app can ask for. */
  available(): Promise<boolean>;
  /** Asks for biometrics or the device passcode; true when the person passed. */
  authenticate(prompt: string): Promise<boolean>;
}

/** The lock setting. It lives in secure storage, never in the database or plain storage. */
export interface LockStore {
  isEnabled(): Promise<boolean>;
  setEnabled(enabled: boolean): Promise<void>;
}

/** True once the app has been in the background for at least `limit` ms. */
export function shouldRelock(backgroundedAt: number | null, now: number, limit: number = LOCK_AFTER_MS): boolean {
  return backgroundedAt !== null && now - backgroundedAt >= limit;
}

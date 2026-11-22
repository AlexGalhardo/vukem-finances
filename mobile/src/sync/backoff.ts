import type { Random } from "./types";

/** Seconds to wait after the 1st, 2nd, 3rd and later failures in a row: 5 s, 15 s, 60 s, 5 min. */
export const BACKOFF_SCHEDULE_MS: readonly number[] = [5_000, 15_000, 60_000, 300_000];

/** How far the wait may stray from the schedule, so phones that failed together do not retry together. */
export const JITTER = 0.2;

/** The wait before the next try after `failures` failures in a row (1 or more), jittered by plus or minus 20 percent. */
export function backoffDelay(failures: number, random: Random): number {
  const base = BACKOFF_SCHEDULE_MS[Math.min(Math.max(failures, 1), BACKOFF_SCHEDULE_MS.length) - 1] ?? 300_000;
  return Math.round(base * (1 - JITTER + 2 * JITTER * random.next()));
}

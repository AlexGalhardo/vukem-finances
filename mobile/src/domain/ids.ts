import { getRandomValues } from "expo-crypto";

const ALPHABET = "0123456789ABCDEFGHJKMNPQRSTVWXYZ";
const TIME_CHARS = 10;
const RANDOM_BYTES = 10;

/** How far ahead of the server clock a client uid may be stamped. */
export const MAX_FUTURE_MS = 24 * 60 * 60 * 1000;

function defaultRandom(): Uint8Array {
  return getRandomValues(new Uint8Array(RANDOM_BYTES));
}

/**
 * A ULID: 48 bits of millisecond time and 80 random bits in 26 upper-case
 * Crockford base32 characters (docs/specs/sync-data-model.md).
 */
export function newUid(now: number = Date.now(), random: () => Uint8Array = defaultRandom): string {
  let time = "";
  let rest = Math.floor(now);
  for (let i = 0; i < TIME_CHARS; i++) {
    time = ALPHABET.charAt(rest % 32) + time;
    rest = Math.floor(rest / 32);
  }

  const bytes = random();
  if (bytes.length < RANDOM_BYTES) {
    throw new RangeError("not enough random bytes");
  }

  // 80 bits as 16 groups of 5 bits, taken from a bit accumulator.
  let rand = "";
  let acc = 0;
  let bits = 0;
  for (const byte of bytes.slice(0, RANDOM_BYTES)) {
    acc = (acc << 8) | byte;
    bits += 8;
    while (bits >= 5) {
      rand += ALPHABET.charAt((acc >> (bits - 5)) & 31);
      bits -= 5;
    }
    acc &= (1 << bits) - 1;
  }

  return time + rand;
}

/** The millisecond timestamp of a valid uid, or null; mirrors the server's ulid.TimeOf. */
export function uidTime(uid: string): number | null {
  if (uid.length !== 26) {
    return null;
  }

  let ms = 0;
  for (let i = 0; i < 26; i++) {
    const v = ALPHABET.indexOf(uid.charAt(i));
    if (v < 0) {
      return null;
    }
    // The first character carries 3 real bits; more would exceed 128 bits.
    if (i === 0 && v > 7) {
      return null;
    }
    if (i < TIME_CHARS) {
      ms = ms * 32 + v;
    }
  }

  return ms;
}

/** Mirrors changes.ValidUID: 26 valid characters and a stamp at most 24 hours ahead. */
export function isValidUid(uid: string, now: number = Date.now()): boolean {
  const at = uidTime(uid);
  return at !== null && at <= now + MAX_FUTURE_MS;
}

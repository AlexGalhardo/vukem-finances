// The X-Webhook-Signature of an AbacatePay delivery, for the billing spec: HMAC-SHA256 of the raw body, base64.
import { createHmac } from "node:crypto";
import { readFileSync } from "node:fs";

// The public key AbacatePay documents for this signature: read from the code that checks it, so they cannot drift.
const publicKey = /pixPublicKey = "([^"]+)"/.exec(readFileSync("app/billing/abacatepay.go", "utf8"))?.[1] ?? "";

export function signPixBody(body) {
  return createHmac("sha256", publicKey).update(body).digest("base64");
}

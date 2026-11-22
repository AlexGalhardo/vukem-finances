import { isValidUid, MAX_FUTURE_MS, newUid, uidTime } from "../../src/domain/ids";

const NOW = Date.UTC(2026, 10, 21, 12, 0, 0);

describe("newUid", () => {
	it("is 26 upper-case Crockford characters", () => {
		expect(newUid()).toMatch(/^[0-7][0-9A-HJKMNP-TV-Z]{25}$/u);
	});

	it("carries its timestamp and passes the server rules", () => {
		const uid = newUid(NOW);
		expect(uidTime(uid)).toBe(NOW);
		expect(isValidUid(uid, NOW)).toBe(true);
	});

	it("encodes known bytes", () => {
		expect(newUid(0, () => new Uint8Array(10))).toBe("0".repeat(26));
		expect(newUid(0, () => new Uint8Array(10).fill(255))).toBe(`${"0".repeat(10)}${"Z".repeat(16)}`);
	});

	it("is random between calls", () => {
		const seen = new Set(Array.from({ length: 200 }, () => newUid(NOW)));
		expect(seen.size).toBe(200);
	});

	it("sorts by time", () => {
		expect(newUid(NOW) < newUid(NOW + 1)).toBe(true);
	});
});

describe("isValidUid", () => {
	it("accepts a stamp up to 24 hours ahead and refuses beyond", () => {
		expect(isValidUid(newUid(NOW + MAX_FUTURE_MS), NOW)).toBe(true);
		expect(isValidUid(newUid(NOW + MAX_FUTURE_MS + 1), NOW)).toBe(false);
	});

	it("refuses bad length, lower case, bad alphabet and overflow", () => {
		const ok = newUid(NOW);
		expect(isValidUid(ok.slice(1), NOW)).toBe(false);
		expect(isValidUid(ok.toLowerCase(), NOW)).toBe(false);
		expect(isValidUid(`I${ok.slice(1)}`, NOW)).toBe(false);
		expect(isValidUid(`8${ok.slice(1)}`, NOW)).toBe(false);
		expect(isValidUid("", NOW)).toBe(false);
	});
});

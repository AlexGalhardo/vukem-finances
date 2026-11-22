import { addMonths } from "../../src/domain/dates";

describe("addMonths", () => {
	it("clamps month ends", () => {
		expect(addMonths("2026-01-31", 1)).toBe("2026-02-28");
		expect(addMonths("2028-01-31", 1)).toBe("2028-02-29");
		expect(addMonths("2026-01-31", 2)).toBe("2026-03-31");
		expect(addMonths("2026-01-31", 3)).toBe("2026-04-30");
	});

	it("does not drift: each result comes from the start date", () => {
		expect(addMonths("2026-01-31", 14)).toBe("2027-03-31");
	});

	it("keeps a leap day on leap years only", () => {
		expect(addMonths("2028-02-29", 12)).toBe("2029-02-28");
		expect(addMonths("2028-02-29", 48)).toBe("2032-02-29");
	});

	it("crosses years both ways", () => {
		expect(addMonths("2026-12-15", 1)).toBe("2027-01-15");
		expect(addMonths("2026-01-15", -1)).toBe("2025-12-15");
		expect(addMonths("2026-03-31", -1)).toBe("2026-02-28");
	});

	it("refuses text that is not a date", () => {
		expect(() => addMonths("2026-1-3", 1)).toThrow(RangeError);
	});
});

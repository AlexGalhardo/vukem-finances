import { isValidTimezone, todayIn } from "../../src/domain/time";

describe("todayIn", () => {
  it("decides today by the workspace time zone", () => {
    expect(todayIn("UTC", new Date(Date.UTC(2027, 2, 15, 23, 30)))).toBe("2027-03-15");
    expect(todayIn("Asia/Tokyo", new Date(Date.UTC(2027, 2, 15, 20, 0)))).toBe("2027-03-16");
    expect(todayIn("America/Sao_Paulo", new Date(Date.UTC(2027, 2, 15, 1, 0)))).toBe("2027-03-14");
  });

  it("falls back to UTC for an unknown zone", () => {
    expect(todayIn("Mars/Olympus", new Date(Date.UTC(2027, 2, 15, 12)))).toBe("2027-03-15");
    expect(isValidTimezone("Mars/Olympus")).toBe(false);
    expect(isValidTimezone("Europe/Lisbon")).toBe(true);
  });
});

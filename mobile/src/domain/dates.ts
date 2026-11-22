/** Calendar dates are ISO "YYYY-MM-DD" strings, never instants (docs/specs/recurrence.md). */

export function daysInMonth(year: number, month: number): number {
	if (month === 2) {
		const leap = (year % 4 === 0 && year % 100 !== 0) || year % 400 === 0;
		return leap ? 29 : 28;
	}
	return [4, 6, 9, 11].includes(month) ? 30 : 31;
}

function pad(n: number, width: number): string {
	return String(n).padStart(width, "0");
}

/**
 * Moves an ISO date by n months and keeps the day of the month, clamped to
 * the last day of the target month: Jan 31 plus one month is Feb 28 (29 in a
 * leap year), never a March date.
 */
export function addMonths(date: string, n: number): string {
	const m = /^(\d{4})-(\d{2})-(\d{2})$/u.exec(date);
	if (m === null) {
		throw new RangeError(`not an ISO date: ${date}`);
	}
	const year = Number(m[1]);
	const month = Number(m[2]);
	const day = Number(m[3]);

	const index = year * 12 + (month - 1) + n;
	const ny = Math.floor(index / 12);
	const nm = (((index % 12) + 12) % 12) + 1;

	return `${pad(ny, 4)}-${pad(nm, 2)}-${pad(Math.min(day, daysInMonth(ny, nm)), 2)}`;
}

const MS_PER_DAY = 86_400_000;

function utcDay(date: string): number {
	const m = /^(\d{4})-(\d{2})-(\d{2})$/u.exec(date);
	if (m === null) {
		throw new RangeError(`not an ISO date: ${date}`);
	}
	return Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])) / MS_PER_DAY;
}

/** Moves an ISO date by n days. */
export function addDays(date: string, n: number): string {
	return new Date((utcDay(date) + n) * MS_PER_DAY).toISOString().slice(0, 10);
}

/** The dates from a to b, both included. */
export function daysBetween(a: string, b: string): number {
	return utcDay(b) - utcDay(a) + 1;
}

/** True for a real calendar date written as YYYY-MM-DD. */
export function isIsoDate(date: string): boolean {
	const m = /^(\d{4})-(\d{2})-(\d{2})$/u.exec(date);
	if (m === null) {
		return false;
	}
	const month = Number(m[2]);
	return month >= 1 && month <= 12 && Number(m[3]) >= 1 && Number(m[3]) <= daysInMonth(Number(m[1]), month);
}

export function monthEnd(date: string): string {
	const y = Number(date.slice(0, 4));
	const mo = Number(date.slice(5, 7));
	return `${date.slice(0, 7)}-${pad(daysInMonth(y, mo), 2)}`;
}

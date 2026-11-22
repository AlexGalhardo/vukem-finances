/** Calendar dates are ISO "YYYY-MM-DD" strings, never instants (docs/specs/recurrence.md). */

function daysInMonth(year: number, month: number): number {
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

import type { Database } from "../db/database";
import type { Row } from "../db/entities";
import type { Repository } from "../db/repository";
import { isIsoDate } from "./dates";
import { type Ledger, LedgerError, accountBalances } from "./ledger";
import { MAX_CENTS } from "./money";
import { cardStatements } from "./statements";

export interface CardInput {
	name: string;
	limit_cents: number;
	closing_day: number;
	due_day: number;
	/** The bank account statements are paid from by default; null means none. */
	payment_account_uid: string | null;
}

function validDay(n: number): boolean {
	return Number.isInteger(n) && n >= 1 && n <= 31;
}

async function checkCard(repo: Repository, input: CardInput): Promise<void> {
	const name = input.name.trim();
	if (name === "" || [...name].length > 80) {
		throw new LedgerError("name", "give the card a name of up to 80 characters");
	}
	if (!Number.isSafeInteger(input.limit_cents) || input.limit_cents <= 0 || input.limit_cents > MAX_CENTS) {
		throw new LedgerError("limit", "the limit must be a positive amount");
	}
	if (!validDay(input.closing_day) || !validDay(input.due_day)) {
		throw new LedgerError("day", "the closing day and the due day must be between 1 and 31");
	}
	if (input.payment_account_uid !== null) {
		const pay = await repo.get("account", input.payment_account_uid);
		if (pay === null || pay.kind === "credit_card") {
			throw new LedgerError("payment", "choose a bank account that is not a credit card to pay from");
		}
	}
}

/** A card is an account of kind credit_card plus its card row, written together. */
export async function createCard(db: Database, repo: Repository, input: CardInput): Promise<Row<"credit_card">> {
	await checkCard(repo, input);
	return db.transaction(async () => {
		const account = await repo.create("account", {
			name: input.name.trim(),
			kind: "credit_card",
			opening_cents: 0,
			archived_at: null,
			position: 0,
		});
		return repo.create("credit_card", {
			account_uid: account.uid,
			limit_cents: input.limit_cents,
			closing_day: input.closing_day,
			due_day: input.due_day,
			payment_account_uid: input.payment_account_uid,
			brand: "",
			last4: "",
		});
	});
}

export interface CardOverview {
	card: Row<"credit_card">;
	name: string;
	/** What the card owes today: the negative balance of its account. */
	used_cents: number;
	available_cents: number;
}

export async function cardOverviews(db: Database, repo: Repository, today: string): Promise<CardOverview[]> {
	const balances = new Map((await accountBalances(db, today)).map((b) => [b.account_uid, b.balance_cents]));
	const accounts = new Map((await repo.list("account")).map((a) => [a.uid, a]));
	const out: CardOverview[] = [];
	for (const card of await repo.list("credit_card")) {
		const account = accounts.get(card.account_uid);
		if (account === undefined) {
			continue;
		}
		const used = Math.max(0, -(balances.get(card.account_uid) ?? 0));
		out.push({ card, name: account.name, used_cents: used, available_cents: Math.max(0, card.limit_cents - used) });
	}
	return out;
}

export interface PaymentInput {
	from_account_uid: string;
	/** null pays what is left of the statement. */
	amount_cents: number | null;
	date: string;
}

/**
 * Pays a statement of a card from a bank account. The payment is a transfer
 * (neither income nor expense), and its card-side leg carries the statement.
 * A partial or extra payment is allowed.
 */
export async function payStatement(
	db: Database,
	repo: Repository,
	ledger: Ledger,
	cardUid: string,
	ref: string,
	input: PaymentInput,
	today: string,
): Promise<void> {
	if (!isIsoDate(input.date)) {
		throw new LedgerError("date", "the date must be a real date");
	}
	await db.transaction(async () => {
		const card = await repo.get("credit_card", cardUid);
		if (card === null) {
			throw new LedgerError("not_found", "card not found");
		}
		const statement = (await cardStatements(db, card, today)).find((s) => s.ref === ref);
		if (statement === undefined) {
			throw new LedgerError("statement", "that statement does not exist");
		}
		const from = await repo.get("account", input.from_account_uid);
		if (from === null) {
			throw new LedgerError("not_found", "account not found");
		}
		if (from.kind === "credit_card") {
			throw new LedgerError("pay_from_card", "a card cannot pay a card");
		}
		let amount = input.amount_cents;
		if (amount === null) {
			if (statement.remaining_cents <= 0) {
				throw new LedgerError("nothing_owed", "nothing is owed on this statement");
			}
			amount = statement.remaining_cents;
		}
		const made = await ledger.createTransfer({
			from_account_uid: input.from_account_uid,
			to_account_uid: card.account_uid,
			amount_cents: amount,
			date: input.date,
			description: `Credit card payment ${ref}`,
		});
		await repo.update("transaction", made.destination, { statement_ref: ref });
	});
}

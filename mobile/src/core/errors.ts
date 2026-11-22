import type { Ui } from "../components/ui";
import { LedgerError } from "../domain/ledger";

/** The message a person sees for a failed write: a known rule gets its own text, anything else the generic one. */
export function describeError(ui: Ui, error: unknown): string {
	if (error instanceof LedgerError) {
		switch (error.code) {
			case "children":
				return ui.t("categoryHasChildren");
			case "amount":
			case "limit":
				return ui.t("invalidAmount");
			case "date":
				return ui.t("invalidDate");
			case "name":
				return ui.t("required");
		}
	}
	return ui.t("errorGeneric");
}

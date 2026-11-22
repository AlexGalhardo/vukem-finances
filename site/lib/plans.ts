import plansJson from "../generated/plans.json";

export interface PlanLimits {
	plan: string;
	accounts: number;
	transactionsPerMonth: number;
	members: number;
	csvImport: boolean;
	budgets: boolean;
}

// -1 is `workspaces.Unlimited` in the app.
export const UNLIMITED = -1;

export const plans: { free: PlanLimits; pro: PlanLimits } = plansJson;

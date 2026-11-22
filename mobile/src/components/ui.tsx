import { createContext, type ReactNode, useContext, useMemo } from "react";
import { type Locale, type MessageKey, translate, type Variables } from "../i18n";
import { spokenMoney } from "../i18n/spoken";
import { colors, fontSize, radius, spacing } from "../theme/tokens";
import { formatMoney } from "../domain/money";

export type Scheme = "light" | "dark";

/** Every touchable is at least this big, in density-independent pixels (the spec asks for 48). */
export const TOUCH_MIN = 48;

export interface Ui {
	locale: Locale;
	scheme: Scheme;
	currency: string;
	colors: (typeof colors)[Scheme];
	spacing: typeof spacing;
	fontSize: typeof fontSize;
	radius: number;
	t: (key: MessageKey, variables?: Variables) => string;
	money: (cents: number) => string;
	spoken: (cents: number) => string;
}

function build(locale: Locale, scheme: Scheme, currency: string): Ui {
	return {
		locale,
		scheme,
		currency,
		colors: colors[scheme],
		spacing,
		fontSize,
		radius,
		t: (key, variables) => translate(locale, key, variables),
		money: (cents) => formatMoney(cents, currency, locale),
		spoken: (cents) => spokenMoney(cents, currency, locale),
	};
}

const UiContext = createContext<Ui>(build("en", "light", "USD"));

export function UiProvider(props: { locale: Locale; scheme: Scheme; currency: string; children: ReactNode }): ReactNode {
	const { locale, scheme, currency, children } = props;
	const value = useMemo(() => build(locale, scheme, currency), [locale, scheme, currency]);
	return <UiContext.Provider value={value}>{children}</UiContext.Provider>;
}

export function useUi(): Ui {
	return useContext(UiContext);
}

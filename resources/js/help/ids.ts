import type help from "../i18n/en/help";

type Keys = keyof typeof help;

/** Every help entry: `hint<Name>` for a field or figure, `term<Name>` for a glossary word. */
export type HelpId = Keys extends infer K ? (K extends `${infer Id}Title` ? Id : never) : never;

/** The glossary words, for TermHint. */
export type TermId = Extract<HelpId, `term${string}`>;

export const helpTitleKey = (id: HelpId): `${HelpId}Title` => `${id}Title`;
export const helpBodyKey = (id: HelpId): `${HelpId}Body` => `${id}Body`;

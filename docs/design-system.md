# Design system

The specification of the vukem-finances interface: tokens, layouts, components, contextual help, keyboard hints, the tutorial and the migration plan. It is written so that an engineer can implement any part without another design pass.

**Status:** specification. Nothing in `resources/` has changed yet. Until task T01 of the [migration plan](#16-migration-plan) lands, `docs/design.md` and `resources/css/tokens.css` still describe the running app.

## Contents

1. [Scope and sources](#1-scope-and-sources)
2. [Stack decisions](#2-stack-decisions)
3. [Principles](#3-principles)
4. [Color](#4-color)
5. [Typography](#5-typography)
6. [Spacing, radius, borders, elevation](#6-spacing-radius-borders-elevation)
7. [Grid and page layouts](#7-grid-and-page-layouts)
8. [Components](#8-components)
9. [Icons](#9-icons)
10. [Motion](#10-motion)
11. [Responsive rules](#11-responsive-rules)
12. [Accessibility](#12-accessibility)
13. [Tooltips and contextual help](#13-tooltips-and-contextual-help)
14. [Keyboard hints on buttons](#14-keyboard-hints-on-buttons)
15. [The tutorial](#15-the-tutorial)
16. [Migration plan](#16-migration-plan)
17. [Open decisions](#17-open-decisions)
18. [Not verified](#18-not-verified)

## 1. Scope and sources

### 1.1 What this refactor is

A full visual and interaction refactor of the React client: restrained, spacious, monochrome surfaces with one green accent, dark and light themes, one icon library, on-screen explanations, visible keyboard shortcuts and a step-by-step tutorial. Routes, page props, permissions, money handling and server behavior do not change.

### 1.2 Originality boundary

The developer picked a well-known developer-tool website as a reference for quality. Decisions D-011 and D-012 still apply, so this document records the **qualities** taken from that study and none of its material:

| Taken (quality) | How it appears here |
| --- | --- |
| Restraint | Near-monochrome surfaces; color is reserved for meaning |
| Spacing rhythm | A strict 4 px scale, with generous space between groups |
| Type hierarchy | Few sizes, tight tracking on headings, quiet labels |
| Form layout | A single narrow column, stacked labels, large fields on auth pages |
| Density | Compact tables inside calm page margins |
| Motion | Short fades, nothing decorative |

Never reproduce another product's name, logo, typefaces, illustrations, copy, screenshots, code or page composition. The product keeps its own mark, its own green and its own numeric, finance-first layouts. Reviewers reject a screen that could be mistaken for another product.

### 1.3 Sources in this repository

`AGENTS.md`, `.agents/rules/`, `docs/design.md`, `docs/shortcuts.md`, `resources/css/`, `resources/js/{components,layouts,pages,shortcuts,i18n}/`, `resources/js/nav.ts`, `routes/`, `app/pages/`, `package.json`.

### 1.4 Current problems this spec fixes

| Problem | Evidence |
| --- | --- |
| The declared font is never loaded | `finance.css` sets `'Geist Variable'`, but no package or `@font-face` provides it, so text falls back to the system sans-serif |
| Two token systems overlap | `finance.css` defines shadcn defaults in `oklch`, then `tokens.css` overrides them |
| The page body ignores the tokens | `resources/views/app.html` hardcodes `bg-white text-zinc-900 dark:bg-zinc-950 dark:text-zinc-100` |
| The green is tinted and the neutrals are green-gray | `--vf-brand: #1c7448`, `--vf-bg: #fbfcfb` |
| Tooltips open instantly | `TooltipProvider` has `delayDuration = 0` and `sideOffset = 0` |
| Native `title` is used as a tooltip | Icon buttons in `Shell.tsx` and `Preferences.tsx`; not available on touch or to keyboard users |
| `Kbd` misses the tooltip contract | `tooltip.tsx` styles `[data-slot=kbd]`, but `Kbd` renders no `data-slot` |
| Hardcoded English | `Auth/TwoFactor.tsx` passes the literal title `Two-factor authentication` |
| `transition-all` on every button | `ui/button.tsx` |
| Theme toggle has two states and ignores live system changes | `Preferences.tsx`, inline script in `app.html` |
| Flat sidebar of 16 items | `nav.ts` has no grouping |

## 2. Stack decisions

### 2.1 Installed versions this spec targets

| Package | Version |
| --- | --- |
| `react`, `react-dom` | 19.3.1 |
| `tailwindcss`, `@tailwindcss/vite` | 4.3.5 |
| `radix-ui` | 1.8.1 |
| `lucide-react` | 1.72.0 |
| `class-variance-authority` | 0.7.1 |
| `tailwind-merge` | 3.7.0 |
| `@inertiajs/react` | 3.11.0 |

### 2.2 shadcn/ui

`resources/js/components/ui/` already follows the shadcn structure on the unified `radix-ui` package, but the repository has no `components.json`. Task T02 adds one, so components can be pulled with an exact CLI version:

```json
{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "new-york",
  "rsc": false,
  "tsx": true,
  "tailwind": {
    "config": "",
    "css": "resources/css/app.css",
    "baseColor": "neutral",
    "cssVariables": true
  },
  "iconLibrary": "lucide",
  "aliases": {
    "components": "resources/js/components",
    "ui": "resources/js/components/ui",
    "lib": "resources/js/lib",
    "utils": "resources/js/lib/utils",
    "hooks": "resources/js/hooks"
  }
}
```

Run `bunx shadcn@4.23.0 add <component>`, never an unpinned tag. Generated code is a starting point: restyle it to this document before committing, and check that the CLI did not edit `tokens.css`. The repository imports with relative paths, so confirm the generated imports resolve (see [section 18](#18-not-verified)).

### 2.3 ui-sdk.dev verdict: do not install now

| Question | Finding |
| --- | --- |
| What it is | UI SDK, a library of headless UI **behaviors** (focus, selection, popups, dismissal, drag and drop, ARIA) with ready-made components, published by Vercel Labs |
| Packages | `@ui-sdk/core` plus a framework adapter such as `@ui-sdk/react` |
| License | Apache-2.0 |
| React | Peer `react >=19`, `react-dom >=19`, `@types/react >=19.2.10`: **compatible** with 19.3.1 and 19.3.0 |
| Tailwind | No Tailwind peer dependency, so it does not conflict with 4.3.5 |
| shadcn | No integration. It is an alternative behavior layer to the Radix primitives under shadcn; it overlaps Dialog, Tooltip, Popover, Select, Tabs, Menu, Combobox, Command and Toast |
| Stability | **Beta only.** All 14 published versions of `@ui-sdk/react` are pre-releases. The npm `latest` tag points at `0.1.0-beta.13`, so a plain install silently takes a beta |

The project forbids beta dependencies, and two behavior layers would mean two focus and dismissal models in one app. **Decision:** keep shadcn on `radix-ui`, which already exports every primitive this spec needs. Revisit when `@ui-sdk/react` publishes a stable `1.x`, and then adopt it only for a behavior Radix lacks. Its Apache-2.0 docs remain a useful reference for interaction patterns.

Sources: <https://ui-sdk.dev/docs>, <https://github.com/vercel-labs/ui>, <https://registry.npmjs.org/@ui-sdk/react>.

### 2.4 Dependencies the refactor needs

| Package | Exact version | License | Type | Why |
| --- | --- | --- | --- | --- |
| `@fontsource-variable/geist` | 5.3.0 | OFL-1.1 | runtime | UI typeface, self-hosted |
| `@fontsource-variable/geist-mono` | 5.3.0 | OFL-1.1 | runtime | Tokens, codes, keys |
| `shadcn` | 4.23.0 | MIT | `bunx` only, not installed | Component scaffolding |

No other dependency is needed. Explicitly **not** added: `cmdk` (`CommandPalette` and `shortcuts/fuzzy.ts` exist), `sonner` (`Toast` exists), `recharts` (charts are in-house SVG), `next-themes` (the inline script is enough), `react-hook-form` (Inertia `useForm`), any tour library (section 15), any second icon set.

## 3. Principles

1. **Numbers first.** The largest, highest-contrast element of a finance screen is the figure the person came to read.
2. **Monochrome by default, green on purpose.** White, black and neutral grays carry the interface. Green marks the brand, the one key action of a view, and positive state.
3. **One primary action per view.** Everything else is secondary, ghost or in a menu.
4. **Space is the separator.** Prefer whitespace and hairlines over boxes inside boxes. Never nest a card in a card.
5. **Explain in place.** Every non-obvious number, term and control has a short explanation one interaction away.
6. **Never color alone.** Money, status and charts always carry a sign, word, icon or pattern.
7. **Keyboard is first-class.** Every shortcut is visible next to the control it triggers.
8. **Quiet motion.** Motion confirms a change; it never decorates.
9. **Every state is designed.** Default, hover, focus, active, disabled, loading, empty and error.
10. **Its own identity.** Inspiration stops at qualities (see [1.2](#12-originality-boundary)).

## 4. Color

### 4.1 The brand color

The brand is exactly `rgb(0, 158, 0)`, `#009e00`, in both themes.

Measured with the WCAG 2.x formula, it has **3.56:1 on white**, 3.26:1 on the light muted surface, 5.89:1 on black and 5.02:1 on the dark muted surface (rounded down). So in the light theme it passes for icons, borders and large text, and **fails for normal text**. White text on it is also 3.56:1, which fails. Black text on it is 5.89:1, which passes.

Rules that follow from this:

| Use | Light theme | Dark theme |
| --- | --- | --- |
| Filled brand button, its label and icon | `--vf-brand` fill, **black** label | same |
| Brand-colored text and links | `--vf-brand-text` `#007a00` | `--vf-brand-text` `#2fc42f` |
| Links | `--vf-brand-text` **and underlined** in running text | same |
| Focus ring | `--vf-focus` `#007a00` | `--vf-focus` `#2fc42f` |
| Icons, chart marks, progress fill, checked controls | `--vf-brand` (3:1 is enough) | same |
| White text on green | **Forbidden** on `#009e00`; allowed only on `--vf-brand-text` in the light theme | Forbidden |
| Normal-size text in `#009e00` on a light surface | **Forbidden** | allowed, but use `--vf-brand-text` |

### 4.2 Tokens

Replace the two `:root` and `.dark` token blocks of `resources/css/tokens.css` with the following. Values stay hex, because `resources/js/design/contrast.test.ts` parses them. Token names must not contain digits, since the test's regular expression is `--(vf-[a-z-]+)`.

```css
:root {
  color-scheme: light;

  --vf-bg: #ffffff;
  --vf-surface: #ffffff;
  --vf-surface-muted: #f5f5f5;
  --vf-border: #e5e5e5;
  --vf-control-border: #858585;

  --vf-text: #0a0a0a;
  --vf-text-muted: #525252;
  --vf-text-subtle: #6b6b6b;
  --vf-text-disabled: #a3a3a3;

  --vf-ink: #0a0a0a;
  --vf-on-ink: #ffffff;

  --vf-brand: #009e00;
  --vf-brand-hover: #00b300;
  --vf-brand-active: #008f00;
  --vf-on-brand: #000000;
  --vf-brand-text: #007a00;
  --vf-brand-subtle: #e8f5e8;

  --vf-income: #007a00;
  --vf-expense: #b42318;
  --vf-on-expense: #ffffff;
  --vf-expense-subtle: #fdecea;

  --vf-warning: #8f5600;
  --vf-warning-subtle: #fff4e0;
  --vf-info: #1d5fc4;
  --vf-info-subtle: #e8f0fc;
  --vf-focus: #007a00;

  --vf-chart-green: #009e00;
  --vf-chart-blue: #2f6fde;
  --vf-chart-amber: #c77700;
  --vf-chart-violet: #8a4fd1;
  --vf-chart-pink: #c23b7a;

  --vf-cat-slate: #566574;
  --vf-cat-blue: #2563b8;
  --vf-cat-teal: #0b7a75;
  --vf-cat-green: #1f7a3d;
  --vf-cat-olive: #6b7a12;
  --vf-cat-amber: #9a6200;
  --vf-cat-orange: #b3541a;
  --vf-cat-red: #c03030;
  --vf-cat-pink: #b83280;
  --vf-cat-violet: #7a4fc4;
}

.dark {
  color-scheme: dark;

  --vf-bg: #000000;
  --vf-surface: #0a0a0a;
  --vf-surface-muted: #171717;
  --vf-border: #262626;
  --vf-control-border: #767676;

  --vf-text: #f5f5f5;
  --vf-text-muted: #a3a3a3;
  --vf-text-subtle: #8f8f8f;
  --vf-text-disabled: #525252;

  --vf-ink: #f5f5f5;
  --vf-on-ink: #0a0a0a;

  --vf-brand: #009e00;
  --vf-brand-hover: #00b300;
  --vf-brand-active: #008f00;
  --vf-on-brand: #000000;
  --vf-brand-text: #2fc42f;
  --vf-brand-subtle: #0b2a0b;

  --vf-income: #2fc42f;
  --vf-expense: #ff8f86;
  --vf-on-expense: #000000;
  --vf-expense-subtle: #2a0f0d;

  --vf-warning: #f0b94a;
  --vf-warning-subtle: #2a1e05;
  --vf-info: #7ab0ff;
  --vf-info-subtle: #0c1d36;
  --vf-focus: #2fc42f;

  --vf-chart-green: #009e00;
  --vf-chart-blue: #4f8df0;
  --vf-chart-amber: #b87a1e;
  --vf-chart-violet: #a078e8;
  --vf-chart-pink: #e0609c;

  --vf-cat-slate: #9aabb8;
  --vf-cat-blue: #7ab0ff;
  --vf-cat-teal: #4fd1c5;
  --vf-cat-green: #6fdc9f;
  --vf-cat-olive: #bfd35a;
  --vf-cat-amber: #f0b94a;
  --vf-cat-orange: #ff9d5c;
  --vf-cat-red: #ff8f86;
  --vf-cat-pink: #ff8ccf;
  --vf-cat-violet: #b79cff;
}
```

### 4.3 Mapping to shadcn tokens

```css
:root,
.dark {
  --background: var(--vf-bg);
  --foreground: var(--vf-text);
  --card: var(--vf-surface);
  --card-foreground: var(--vf-text);
  --popover: var(--vf-surface);
  --popover-foreground: var(--vf-text);
  --primary: var(--vf-ink);
  --primary-foreground: var(--vf-on-ink);
  --secondary: var(--vf-surface-muted);
  --secondary-foreground: var(--vf-text);
  --muted: var(--vf-surface-muted);
  --muted-foreground: var(--vf-text-muted);
  --accent: var(--vf-surface-muted);
  --accent-foreground: var(--vf-text);
  --destructive: var(--vf-expense);
  --destructive-foreground: var(--vf-on-expense);
  --border: var(--vf-border);
  --input: var(--vf-control-border);
  --ring: var(--vf-focus);
  --radius: var(--vf-radius);

  --brand: var(--vf-brand);
  --brand-hover: var(--vf-brand-hover);
  --brand-active: var(--vf-brand-active);
  --brand-foreground: var(--vf-on-brand);
  --brand-text: var(--vf-brand-text);
  --brand-subtle: var(--vf-brand-subtle);

  --chart-1: var(--vf-chart-green);
  --chart-2: var(--vf-chart-blue);
  --chart-3: var(--vf-chart-amber);
  --chart-4: var(--vf-chart-violet);
  --chart-5: var(--vf-chart-pink);

  --sidebar: var(--vf-bg);
  --sidebar-foreground: var(--vf-text-muted);
  --sidebar-primary: var(--vf-brand);
  --sidebar-primary-foreground: var(--vf-on-brand);
  --sidebar-accent: var(--vf-surface-muted);
  --sidebar-accent-foreground: var(--vf-text);
  --sidebar-border: var(--vf-border);
  --sidebar-ring: var(--vf-focus);
}
```

`finance.css` keeps only its `@theme inline` block, with these additions, and loses its `oklch` `:root` and `.dark` blocks:

```css
@theme inline {
  --font-sans: "Geist Variable", ui-sans-serif, system-ui, sans-serif;
  --font-mono: "Geist Mono Variable", ui-monospace, monospace;
  --font-heading: var(--font-sans);
  --color-brand: var(--brand);
  --color-brand-hover: var(--brand-hover);
  --color-brand-active: var(--brand-active);
  --color-brand-foreground: var(--brand-foreground);
  --color-brand-text: var(--brand-text);
  --color-brand-subtle: var(--brand-subtle);
  --color-destructive-foreground: var(--destructive-foreground);
  --radius-sm: 0.25rem;
  --radius-md: 0.375rem;
  --radius-lg: 0.5rem;
  --radius-xl: 0.75rem;
}
```

### 4.4 Measured contrast

Computed with the WCAG 2.x relative-luminance formula from the token values above. Required: 4.5:1 for text, 3:1 for non-text. Every row passes.

| Pair | Use | Min | Light colors | Light | Dark colors | Dark |
| --- | --- | --- | --- | --- | --- | --- |
| `text` on `bg` | text | 4.5:1 | #0a0a0a / #ffffff | **19.79** | #f5f5f5 / #000000 | **19.26** |
| `text` on `surface` | text | 4.5:1 | #0a0a0a / #ffffff | **19.79** | #f5f5f5 / #0a0a0a | **18.15** |
| `text` on `surface-muted` | text | 4.5:1 | #0a0a0a / #f5f5f5 | **18.15** | #f5f5f5 / #171717 | **16.44** |
| `text-muted` on `bg` | text | 4.5:1 | #525252 / #ffffff | **7.81** | #a3a3a3 / #000000 | **8.32** |
| `text-muted` on `surface` | text | 4.5:1 | #525252 / #ffffff | **7.81** | #a3a3a3 / #0a0a0a | **7.84** |
| `text-muted` on `surface-muted` | text | 4.5:1 | #525252 / #f5f5f5 | **7.16** | #a3a3a3 / #171717 | **7.10** |
| `text-subtle` on `bg` | text | 4.5:1 | #6b6b6b / #ffffff | **5.32** | #8f8f8f / #000000 | **6.49** |
| `text-subtle` on `surface` | text | 4.5:1 | #6b6b6b / #ffffff | **5.32** | #8f8f8f / #0a0a0a | **6.12** |
| `text-subtle` on `surface-muted` | text | 4.5:1 | #6b6b6b / #f5f5f5 | **4.88** | #8f8f8f / #171717 | **5.54** |
| `brand-text` on `bg` | text | 4.5:1 | #007a00 / #ffffff | **5.54** | #2fc42f / #000000 | **9.05** |
| `brand-text` on `surface` | text | 4.5:1 | #007a00 / #ffffff | **5.54** | #2fc42f / #0a0a0a | **8.53** |
| `brand-text` on `surface-muted` | text | 4.5:1 | #007a00 / #f5f5f5 | **5.09** | #2fc42f / #171717 | **7.73** |
| `income` on `bg` | text | 4.5:1 | #007a00 / #ffffff | **5.54** | #2fc42f / #000000 | **9.05** |
| `income` on `surface` | text | 4.5:1 | #007a00 / #ffffff | **5.54** | #2fc42f / #0a0a0a | **8.53** |
| `income` on `surface-muted` | text | 4.5:1 | #007a00 / #f5f5f5 | **5.09** | #2fc42f / #171717 | **7.73** |
| `expense` on `bg` | text | 4.5:1 | #b42318 / #ffffff | **6.57** | #ff8f86 / #000000 | **9.52** |
| `expense` on `surface` | text | 4.5:1 | #b42318 / #ffffff | **6.57** | #ff8f86 / #0a0a0a | **8.98** |
| `expense` on `surface-muted` | text | 4.5:1 | #b42318 / #f5f5f5 | **6.03** | #ff8f86 / #171717 | **8.13** |
| `warning` on `bg` | text | 4.5:1 | #8f5600 / #ffffff | **6.00** | #f0b94a / #000000 | **11.74** |
| `warning` on `surface` | text | 4.5:1 | #8f5600 / #ffffff | **6.00** | #f0b94a / #0a0a0a | **11.07** |
| `warning` on `surface-muted` | text | 4.5:1 | #8f5600 / #f5f5f5 | **5.50** | #f0b94a / #171717 | **10.02** |
| `info` on `bg` | text | 4.5:1 | #1d5fc4 / #ffffff | **6.02** | #7ab0ff / #000000 | **9.48** |
| `info` on `surface` | text | 4.5:1 | #1d5fc4 / #ffffff | **6.02** | #7ab0ff / #0a0a0a | **8.93** |
| `info` on `surface-muted` | text | 4.5:1 | #1d5fc4 / #f5f5f5 | **5.52** | #7ab0ff / #171717 | **8.09** |
| `control-border` on `bg` | non-text | 3:1 | #858585 / #ffffff | **3.69** | #767676 / #000000 | **4.62** |
| `control-border` on `surface` | non-text | 3:1 | #858585 / #ffffff | **3.69** | #767676 / #0a0a0a | **4.35** |
| `control-border` on `surface-muted` | non-text | 3:1 | #858585 / #f5f5f5 | **3.38** | #767676 / #171717 | **3.94** |
| `focus` on `bg` | non-text | 3:1 | #007a00 / #ffffff | **5.54** | #2fc42f / #000000 | **9.05** |
| `focus` on `surface` | non-text | 3:1 | #007a00 / #ffffff | **5.54** | #2fc42f / #0a0a0a | **8.53** |
| `focus` on `surface-muted` | non-text | 3:1 | #007a00 / #f5f5f5 | **5.09** | #2fc42f / #171717 | **7.73** |
| `brand` on `bg` | non-text | 3:1 | #009e00 / #ffffff | **3.56** | #009e00 / #000000 | **5.89** |
| `brand` on `surface` | non-text | 3:1 | #009e00 / #ffffff | **3.56** | #009e00 / #0a0a0a | **5.55** |
| `brand` on `surface-muted` | non-text | 3:1 | #009e00 / #f5f5f5 | **3.26** | #009e00 / #171717 | **5.02** |
| `on-brand` on `brand` | button label | 4.5:1 | #000000 / #009e00 | **5.89** | #000000 / #009e00 | **5.89** |
| `on-brand` on `brand-hover` | button label | 4.5:1 | #000000 / #00b300 | **7.44** | #000000 / #00b300 | **7.44** |
| `on-brand` on `brand-active` | button label | 4.5:1 | #000000 / #008f00 | **4.92** | #000000 / #008f00 | **4.92** |
| `on-ink` on `ink` | button label | 4.5:1 | #ffffff / #0a0a0a | **19.79** | #0a0a0a / #f5f5f5 | **18.15** |
| `on-expense` on `expense` | button label | 4.5:1 | #ffffff / #b42318 | **6.57** | #000000 / #ff8f86 | **9.52** |
| `brand-text` on `brand-subtle` | badge | 4.5:1 | #007a00 / #e8f5e8 | **4.93** | #2fc42f / #0b2a0b | **6.70** |
| `expense` on `expense-subtle` | badge | 4.5:1 | #b42318 / #fdecea | **5.74** | #ff8f86 / #2a0f0d | **8.12** |
| `warning` on `warning-subtle` | badge | 4.5:1 | #8f5600 / #fff4e0 | **5.50** | #f0b94a / #2a1e05 | **9.12** |
| `info` on `info-subtle` | badge | 4.5:1 | #1d5fc4 / #e8f0fc | **5.24** | #7ab0ff / #0c1d36 | **7.61** |

Category colors (`--vf-cat-*`, unchanged, non-text): the lowest ratio across all ten colors and three surfaces is **4.36** in light and **7.59** in dark (required 3:1).

Reference: `#009e00` on white 3.56:1, white on `#009e00` 3.56:1, black on `#009e00` 5.89:1. Ratios are rounded down.

Disabled controls are exempt from WCAG 1.4.3, and `--vf-text-disabled` is deliberately below 4.5:1. A disabled control must still show its label, and the reason in a tooltip when the reason is not obvious.

T01 changes `contrast.test.ts`: move `vf-brand` from `textOnSurface` to `nonText`, and add `vf-brand-text`, `vf-text-subtle`, the status tokens and the on-color pairs of this table.

### 4.5 Theme behavior

- Three states: **System** (no stored value, the default), **Light**, **Dark**. Keep the storage key `theme` and the `.dark` class on `<html>`.
- The control lives in the user menu as a three-option radio group; System shows the resolved value, for example "System (dark)". The header no longer has a permanent toggle.
- The inline script in `app.html` stays first in `<head>`, keeps its CSP nonce and `try/catch`.
- A small client module listens to `matchMedia("(prefers-color-scheme: dark)")` while no value is stored, and to the `storage` event to follow other tabs.
- Choosing System **removes** the stored value.
- The base `:root` token block is the light theme. When JavaScript does not run, the page stays light and readable.
- Remove the `bg-white text-zinc-900 dark:bg-zinc-950 dark:text-zinc-100` classes from `<body>`; `@layer base` provides the colors.

### 4.6 Money and status

| Meaning | Text token | Required non-color signal |
| --- | --- | --- |
| Income | `--vf-income` | `+` sign and, in lists, `ArrowDownLeft` icon; `aria-label` "plus ..." |
| Expense | `--vf-expense` | `−` (U+2212) sign and `ArrowUpRight` icon; `aria-label` "minus ..." |
| Transfer | `--vf-text-muted` | `ArrowLeftRight` icon and the word "Transfer" |
| Zero | `--vf-text-muted` | no sign |
| Success | `--vf-brand-text` | `CircleCheck` and a word |
| Warning | `--vf-warning` | `TriangleAlert` and a word |
| Error | `--vf-expense` | `CircleAlert` and a word |
| Info | `--vf-info` | `Info` and a word |

### 4.7 Chart colors

Categorical order is fixed and never cycled: green, blue, amber, violet, pink. Validated with the `dataviz` palette validator against `#ffffff` and `#f5f5f5` (light) and `#000000`, `#0a0a0a` and `#171717` (dark): every check passes (lightness band, chroma, CVD separation, normal-vision floor, 3:1 contrast).

| Slot | Token | Light | Lowest on light surfaces | Dark | Lowest on dark surfaces |
| --- | --- | --- | --- | --- | --- |
| `--chart-1` | `--vf-chart-green` | #009e00 | 3.26 | #009e00 | 5.02 |
| `--chart-2` | `--vf-chart-blue` | #2f6fde | 4.32 | #4f8df0 | 5.46 |
| `--chart-3` | `--vf-chart-amber` | #c77700 | 3.17 | #b87a1e | 4.98 |
| `--chart-4` | `--vf-chart-violet` | #8a4fd1 | 4.67 | #a078e8 | 5.41 |
| `--chart-5` | `--vf-chart-pink` | #c23b7a | 4.58 | #e0609c | 5.39 |

Rules:

- A chart of categories uses the category's own `--vf-cat-*` color, so color follows the entity. `--chart-*` is for series that are not categories.
- Income and expense in one chart: income is `--vf-income`, solid; expense is `--vf-expense` with a 45-degree hatch pattern defined once as an SVG `<pattern>`. Red and green are never told apart by hue alone.
- "Previous period" is a neutral dashed line in `--vf-text-muted`, not a categorical slot.
- More than 5 series fold into "Other". One y-axis only.
- Marks: 2 px lines, bars with a 2 px gap and a 4 px top radius, markers at least 8 px.
- Grid lines use `--vf-border`; axis labels use `--vf-text-muted` at `text-xs` with `tabular-nums`.
- Value labels use text tokens, never the series color.
- Keep `ChartWithTable`: the chart is `aria-hidden`, the table is the accessible view, and print shows the table.
- In forced-colors mode, series use `CanvasText` with distinct dash and hatch patterns.

## 5. Typography

### 5.1 Typefaces

| Role | Family | Package | License |
| --- | --- | --- | --- |
| UI and numbers | Geist Variable | `@fontsource-variable/geist` 5.3.0 | SIL Open Font License 1.1 |
| Codes, tokens, API keys | Geist Mono Variable | `@fontsource-variable/geist-mono` 5.3.0 | SIL Open Font License 1.1 |

Geist already was the app's declared intent, is free for commercial use, is self-hosted (no third-party request, CSP unchanged), and has true tabular figures. Measured at 100 px: proportional digits are 66.31, 38.41 and 52.41 wide for `0`, `1`, `7`; with `tnum` all three are 60. Its `latin` subset covers pt-BR and es, and U+2212.

Load in `resources/js/app.tsx`, before the app CSS:

```ts
import "@fontsource-variable/geist/wght.css";
import "@fontsource-variable/geist-mono/wght.css";
```

Keep the default `font-display: swap`. Preload only the Latin `wght` file of Geist if LCP needs it. Include the OFL notice in third-party notices if the release bundles any.

### 5.2 Scale

Root stays `16px`, and sizes are in `rem`. Add these to `tokens.css` as `--vf-text-*` and to `@theme inline` as `--text-*`.

| Token | Size / line height | Weight | Tracking | Use |
| --- | --- | --- | --- | --- |
| `display` | 40 / 48 | 600 | -0.025em | Landing hero (desktop), maximum |
| `hero-number` | 36 / 44 | 600 | -0.025em | The main balance |
| `h1-auth` | 28 / 36 | 600 | -0.02em | Auth page title |
| `h1` | 24 / 32 | 600 | -0.02em | Page title |
| `h2` | 18 / 28 | 600 | -0.01em | Section title |
| `h3` | 15 / 24 | 600 | -0.005em | Card title |
| `body` | 14 / 20 | 400 | 0 | Default in the app |
| `body-lg` | 16 / 24 | 400 | 0 | Auth, tutorial, long help |
| `label` | 14 / 20 | 500 | 0 | Form labels, buttons |
| `caption` | 13 / 18 | 400 | 0 | Help text, table meta |
| `overline` | 12 / 16 | 500 | 0.02em | Sidebar group, table header |
| `kbd` | 11 / 16 | 500 | 0 | Key caps |
| `mono` | 13 / 20 | 400 | 0 | Codes and tokens |

Rules:

- Weights are 400, 500 and 600 only.
- No all-caps text (pt-BR and es read worse, and screen readers may spell it).
- Page copy line length is at most `65ch`.
- Inputs use 16 px on coarse pointers, which prevents iOS zoom, and 14 px on fine pointers.
- Every amount, date column and percentage uses `tabular-nums`. Amounts are right-aligned in tables.
- `text-wrap: balance` on headings, `text-wrap: pretty` on help text.

## 6. Spacing, radius, borders, elevation

### 6.1 Spacing

Base unit 4 px. Keep the existing tokens and add the missing steps:

| Token | px | Tailwind | Typical use |
| --- | --- | --- | --- |
| `--vf-space-1` | 4 | `1` | Icon to text in a tight badge |
| `--vf-space-2` | 8 | `2` | Icon to label, label to input, button group |
| `--vf-space-3` | 12 | `3` | Table cell inline padding, list rows |
| `--vf-space-4` | 16 | `4` | Mobile page gutter, field to field |
| `--vf-space-5` | 20 | `5` | Card padding |
| `--vf-space-6` | 24 | `6` | Between cards, desktop card padding |
| `--vf-space-8` | 32 | `8` | Desktop page gutter, header to content |
| `--vf-space-10` | 40 | `10` | Between page sections |
| `--vf-space-12` | 48 | `12` | Auth title to form on large screens |
| `--vf-space-16` | 64 | `16` | Page top on auth |
| `--vf-space-24` | 96 | `24` | Landing sections |

Inner spacing is always smaller than outer spacing. A label sits 8 px from its input; fields sit 16 px apart; form sections sit 32 px apart.

### 6.2 Radius

| Token | px | Use |
| --- | --- | --- |
| `--radius-sm` | 4 | Kbd, checkbox, chart bars |
| `--radius-md` | 6 | Buttons, inputs, selects, menu items, badges |
| `--radius-lg` | 8 | Cards, popovers, tooltips, toasts |
| `--radius-xl` | 12 | Dialogs, sheets, command palette |
| `full` | 9999 | Avatars, status dots, switches |

Set `--vf-radius: 0.5rem`. Replace the `calc()` radius scale in `finance.css` with the fixed values above.

### 6.3 Borders

- 1 px everywhere. `--vf-border` for decorative hairlines (cards, tables, separators).
- `--vf-control-border` for anything a person must locate to operate: inputs, selects, checkboxes, outline buttons. This is the WCAG 1.4.11 boundary, so a control never relies on `--vf-border`.
- Selected and invalid states change the border color **and** add a non-color signal.

### 6.4 Elevation

Surfaces are flat. Only floating layers get a shadow, and a border is always present, because shadows disappear in forced-colors mode and on black.

| Level | Token | Light | Dark | Use |
| --- | --- | --- | --- | --- |
| 0 | none | none | none | Page, cards |
| 1 | `--vf-shadow-sm` | `0 1px 2px rgb(0 0 0 / 0.06)` | none | Sticky header on scroll |
| 2 | `--vf-shadow-md` | `0 4px 12px rgb(0 0 0 / 0.08)` | `0 4px 12px rgb(0 0 0 / 0.6)` | Popover, menu, tooltip, toast |
| 3 | `--vf-shadow-lg` | `0 16px 48px rgb(0 0 0 / 0.14)` | `0 16px 48px rgb(0 0 0 / 0.7)` | Dialog, sheet, palette |

The dialog backdrop is `rgb(0 0 0 / 0.5)` in light and `rgb(0 0 0 / 0.7)` in dark. No blur.

Z-index: sticky header 10, sidebar 20, overlay 40, dialog and sheet 50, popover and menu 55, tooltip 60, tutorial step 65, toast 70, skip link 80.

## 7. Grid and page layouts

All layouts use flex or grid, with `min-w-0` on flex children that hold tables or truncated text. Use logical properties.

### 7.1 Auth

Used by Login, Register, CheckEmail, VerifyEmail, ForgotPassword, ResetPassword, TwoFactorChallenge, Setup, InviteAccept and InviteRegister. New file `layouts/AuthLayout.tsx`.

```text
+--------------------------------------------------+
| [mark] vukem-finances            [language][theme]|  h-16, px-4 md:px-8
|                                                  |
|              Title (h1-auth)                     |  column max-w-[400px]
|              One line of description             |  pt-16 on md+, pt-8 on mobile
|                                                  |
|              [ Continue with Google ]            |  only when enabled
|              ---------- or ----------            |
|              Email                               |
|              [______________________]            |  h-11
|              Password            Forgot?         |
|              [__________________ eye]            |
|              [       Sign in        ]            |  h-11, brand
|                                                  |
|              No account? Sign up                 |
+--------------------------------------------------+
```

- Page background `--vf-bg`, with no card, border or illustration around the form.
- Column: `mx-auto w-full max-w-[400px] px-4`, `flex flex-col gap-6`.
- Fields: `h-11`, 16 px text, 8 px label gap, 16 px between fields.
- One submit button, full width, `brand`, size `lg`.
- Error summary above the form with `role="alert"`; field errors below each field. Focus moves to the first invalid field.
- `autocomplete`: `email`, `current-password`, `new-password`, `one-time-code`. Pasting is always allowed.
- Password toggle: a real `button` with `aria-pressed` (existing `PasswordField`).
- The two-factor code is one `inputmode="numeric"` field, never several boxes.
- The header sign-up link appears only when the edition has public sign-up. `personal` never shows sign-up or billing wording.

### 7.2 App shell

Implemented in `layouts/Shell.tsx`, keeping `SidebarProvider`, the skip link, `main#content` and the live region.

```text
+------------+-------------------------------------------+
| [mark] name| [=] Workspace / Page   [Search Ctrl K] [+ Add] [?] [bell] [avatar] |
| [workspace]+-------------------------------------------+
|            |                                           |
| Dashboard  |   PageHeader                              |
| Transactions   content, max-w-[1200px]                 |
| ...        |                                           |
| group      |                                           |
| ...        |                                           |
| Help       |                                           |
+------------+-------------------------------------------+
```

- Sidebar 240 px; collapsed rail 56 px with tooltips; below `md` a `Sheet` from the inline start.
- Header 56 px, sticky, `border-b`, background `--vf-bg`. It holds: sidebar trigger, breadcrumb, then search, the **Add transaction** brand button, help menu, notifications and user menu.
- The user menu holds name, theme, language, single-key shortcuts, two-factor and log out. `Preferences` moves here.
- Main: `mx-auto w-full max-w-[1200px] px-4 py-6 md:px-8 md:py-8`, sections `gap-8`.
- The sidebar active item uses `--sidebar-accent`, `--vf-text`, weight 500, a 2 px brand bar at the inline start, and `aria-current="page"`.

Sidebar groups, added to `NavItem` as `group`:

| Group | Items |
| --- | --- |
| (none) | Dashboard, Transactions |
| Money | Accounts, Cards, Recurring |
| Planning | Categories, Budgets, Goals |
| Insights | Reports, Suggestions, AI |
| Data | Import |
| Settings | Workspace, Tokens, Devices, Billing (`saas` only) |

Footer: Admin (only with `workspace.admin`), Help and tutorial.

### 7.3 List

Accounts, Transactions, Recurrences, Tokens, Devices, Notifications, Admin ResourceList.

- `PageHeader` (title, description, primary action).
- Toolbar: search, filters, then secondary actions right-aligned. Wraps on small screens.
- `DataTable` inside a borderless section with a top hairline, sticky header, 44 px rows.
- `Pagination` below. Active filters appear as removable chips.
- Below `md`, rows become stacked rows ([8.4](#84-table)).

### 7.4 Detail

CardStatement, Admin ResourceShow.

- Breadcrumb back link, then `PageHeader` with the entity name and actions.
- Summary row: 2 to 4 `Stat` items, `grid-cols-2 lg:grid-cols-4`.
- Content below; on `lg`, an optional side column of 320 px: `grid lg:grid-cols-[minmax(0,1fr)_320px] gap-8`.

### 7.5 Form

Short forms (at most 6 fields) open in a `Dialog`; forms that need page context (a quick add from the list) use a `Sheet`. Full-page forms use a single column.

- Column `max-w-[560px]`; `flex flex-col gap-4`, sections `gap-8` with an `h2`.
- Labels above inputs. Required fields are marked with text ("Optional" on the minority kind).
- Validate on blur after an edit and on submit. Clear on input. Never disable submit to block validation; disable only while the request is in flight.
- Footer: primary action, then Cancel. In a dialog, align to the inline end; on mobile, stack full width with primary on top.

### 7.6 Dashboard

```text
Total balance (hero-number)      [This month v]
Income +R$ 8.200,00    Expenses −R$ 5.120,00    Net +R$ 3.080,00
--------------------------------------------------
Cash flow (2/3)                | Spending by category (1/3)
--------------------------------------------------
Coming up (1/2)                | Budgets (1/2)
```

- Hero row has no card. It is the highest-contrast element of the screen.
- Grid: `grid gap-6 lg:grid-cols-3`; the chart spans 2 columns.
- Cards: 1 px border, no shadow, 20 to 24 px padding.

### 7.7 Settings

Workspace, AI, Billing, Telegram, TwoFactor. Sections are rows with a description on the left and controls on the right:

```text
Section title          | [controls]
Why this exists, in    | [controls]
one or two lines       | [Save]
-------------------------------------
```

`grid gap-4 lg:grid-cols-[280px_minmax(0,1fr)] lg:gap-8`, sections separated by a hairline and `py-8`. Each section saves independently. Destructive actions go in a final "Danger zone" section with an `--vf-expense` border.

### 7.8 Empty state

Uses `ui/empty.tsx`. Centered in the content area, `max-w-[360px]`, `py-16`:

1. Lucide icon, 24 px, in a 40 px `--vf-surface-muted` square.
2. Title in `h3`: what is missing.
3. One sentence: **why it is useful**.
4. Primary action. An optional "Learn how" link opens this screen's tutorial.

A filtered list with no result is a different state: it says so and offers **Clear filters**.

### 7.9 Error and landing

- `Error`: AuthLayout column, status code in `caption`, a plain title, one sentence, and a link back. Never show a stack or raw error.
- `Home` (`saas`): a landing with its own composition: hero with the product's real dashboard rendered with sample numbers, three feature rows, pricing summary and call to action. Black or white background by theme, with green as the only accent. Remove the current lime `finance-landing` styles.

## 8. Components

Conventions: every component sets `data-slot`; variants use `cva`; `className` merges through `cn`. State classes are written once in the variant. Every interactive component meets [section 12](#12-accessibility).

### 8.1 Button

| Variant | Rest | Hover | Active | Use |
| --- | --- | --- | --- | --- |
| `default` | `--vf-ink` fill, `--vf-on-ink` text | 88% ink | 80% ink | Primary action |
| `brand` | `--vf-brand` fill, **black** text | `--vf-brand-hover` | `--vf-brand-active` | The single key action of a view: auth submit, Add transaction |
| `secondary` | `--vf-surface-muted` | 6% darker or lighter via `color-mix` | 10% | Secondary action |
| `outline` | `--vf-bg`, `--vf-control-border` | `--vf-surface-muted` | same | Toolbar, filters |
| `ghost` | transparent | `--vf-surface-muted` | same | Icon buttons, tertiary |
| `destructive` | `--vf-expense` fill, `--vf-on-expense` text | 90% | 80% | Only inside a confirm dialog |
| `destructive-outline` | transparent, `--vf-expense` text and border | `--vf-expense-subtle` | same | Delete in a page or row |
| `link` | `--vf-brand-text`, underline | thicker underline | same | Inline action |

| Size | Height | Inline padding | Text | Icon |
| --- | --- | --- | --- | --- |
| `xs` | 24 | 8 | 12 | 12 |
| `sm` | 32 | 12 | 13 | 14 |
| `default` | 36 | 14 | 14 | 16 |
| `lg` | 44 | 18 | 16 | 18 |
| `icon-sm`, `icon`, `icon-lg` | 32, 36, 44 | 0 | | 14, 16, 18 |

- Keep `pointer-coarse:min-h-11 pointer-coarse:min-w-11`, so touch targets are 44 px.
- Weight 500, `gap-2`, `rounded-md`.
- States: focus uses the global ring; disabled is `--vf-surface-muted` with `--vf-text-disabled` and no pointer events; loading keeps the width, sets `aria-busy="true"`, disables, and shows a `LoaderCircle` spinner beside the label.
- Replace `transition-all` with `transition-colors`. Remove `active:translate-y-px`.
- Icon-only buttons need `aria-label` and a tooltip.
- New optional prop: `shortcutAction?: string` ([section 14](#14-keyboard-hints-on-buttons)).

### 8.2 Input and textarea

| Property | Value |
| --- | --- |
| Sizes | `sm` 32, `default` 36, `lg` 44 (auth); `pointer-coarse:min-h-11` |
| Rest | `--vf-bg`, `--vf-control-border`, `--vf-text`, `rounded-md`, `px-3` |
| Placeholder | `--vf-text-subtle`, an example only, never the label |
| Hover | border `--vf-text-muted` |
| Focus | global ring, border `--vf-focus` |
| Invalid | `aria-invalid`, `--vf-expense` border, `CircleAlert` and message below linked by `aria-describedby` |
| Disabled | `--vf-surface-muted`, `--vf-text-disabled` |
| Read-only | `--vf-surface-muted`, normal text color |
| Adornments | Leading icon or currency inside, with `ps-9`; trailing action is a real button |

`AmountInput`: right-aligned, `tabular-nums`, currency prefix, `inputmode="decimal"`, `enterkeyhint="next"`. The parsed value keeps going through the existing money path; no floats.

Search fields sit inside `<search>` with `type="search"`.

### 8.3 Select, combobox, checkbox, radio, switch

- `NativeSelect` stays the default for short static lists on forms and on mobile.
- `ui/select.tsx` (Radix, new) for styled single choice with up to about 15 options. Same box as `Input`, `ChevronDown` 16 px, menu at elevation 2, items 32 px (`pointer-coarse:min-h-11`), checked item with `Check`.
- `Combobox` (existing) for long or searchable lists: accounts, categories, tags. Keep `role="combobox"`, which also makes the shortcut guard ignore typing.
- Two to five exclusive options are radios or a segmented control, not a select.
- Checkbox 16 px (hit area 44 on coarse), `--vf-control-border`; checked is a `--vf-brand` fill with a black `Check`.
- Switch 36×20; on is `--vf-brand` with a white thumb bordered by `--vf-control-border`, and the state appears in text beside it ("On", "Off"). Use only for settings that apply immediately.

### 8.4 Table

`DataTable` on `ui/table.tsx`.

| Part | Spec |
| --- | --- |
| Header | 36 px, `overline`, `--vf-text-muted`, bottom hairline, sticky in scroll containers |
| Row | 44 px (`compact` 36), bottom hairline; no zebra |
| Hover | `--vf-surface-muted` |
| Selected | `--vf-brand-subtle`, a checked box and `aria-selected` |
| Focus | ring inset on the row or cell |
| Numbers | Right-aligned, `tabular-nums`, with `MoneyText` |
| Sort | Header is a button with `aria-sort` and an arrow icon |
| Actions | Last column, `MoreHorizontal` ghost icon button opening a menu; primary row action can be inline |
| Loading | Skeleton rows matching the column widths |
| Empty | [7.8](#78-empty-state) inside the table area |

Responsive: below `md`, a table with more than 3 columns renders as a list where each row shows the main text and amount on line one and meta on line two. Column labels remain available to screen readers. Tables that must stay tabular scroll horizontally in a focusable region with `role="region"`, `aria-label` and `tabindex="0"`.

### 8.5 Dialog

Radix Dialog (`ui/dialog.tsx`).

- Widths: `sm` 400, `md` 520, `lg` 680. On phones: full width minus 16 px, at most `100dvh - 32px`, and the body scrolls.
- Padding 24; header `h2` with an optional description; footer aligned to the end.
- `rounded-xl`, elevation 3, backdrop per [6.4](#64-elevation).
- Focus: first field, or the **safe** action in destructive confirmations (`ConfirmDelete`). Focus returns to the opener.
- Esc closes unless the form is dirty; then ask before discarding. Do not close on a backdrop click when dirty.
- `ConfirmDelete` states the object and consequence, and for the worst cases asks to type the name.

### 8.6 Sheet

Radix Dialog variant (`ui/sheet.tsx`): inline-end side, 420 px, or a bottom sheet on phones with a max height of `85dvh`. Used for quick add, filters on mobile, help and the mobile sidebar. Same focus rules as the dialog. Motion: slide 16 px plus a fade; a fade only under reduced motion.

### 8.7 Tabs

`ui/tabs.tsx`, used in Reports.

- `line` variant (default): text tabs, 36 px, `--vf-text-muted`, active has `--vf-text` and a 2 px bottom bar in `--vf-brand`.
- `segmented` variant for 2 to 4 short views: `--vf-surface-muted` track, active is `--vf-bg` with a border.
- Arrow keys move, `Home` and `End` jump. When tabs overflow, the list scrolls horizontally and the active tab stays in view.
- Tabs in a page reflect the URL when the view is shareable.

### 8.8 Card

- `--vf-surface`, 1 px `--vf-border`, `rounded-lg`, no shadow.
- Padding: 20 on mobile, 24 on `md`. Header `h3` with an optional action on the right.
- Variants: `default`, `flat` (no border, for grouped sections), `interactive` (whole card is a link, hover `--vf-surface-muted`, one focus ring).
- `Stat` (new): label in `caption`, optional `HelpHint`, value in `h2` or `hero-number`, optional delta with sign and arrow.

### 8.9 Badge

Height 20, `px-2`, `text-xs`, weight 500, `rounded-md`.

| Variant | Background | Text | Use |
| --- | --- | --- | --- |
| `neutral` | `--vf-surface-muted` | `--vf-text` | Default, plan |
| `outline` | transparent | `--vf-text-muted` | Metadata |
| `success` | `--vf-brand-subtle` | `--vf-brand-text` | Paid, active |
| `warning` | `--vf-warning-subtle` | `--vf-warning` | Closing soon, paused |
| `danger` | `--vf-expense-subtle` | `--vf-expense` | Overdue, failed |
| `info` | `--vf-info-subtle` | `--vf-info` | Trial, new |

A status badge always carries a word, plus a 12 px icon when it is the only signal in the row. `cards/StatusBadge.tsx` adopts these variants.

### 8.10 Toast

Keep `components/Toast.tsx`; restyle it.

- Bottom-end on desktop, bottom full width on phones above the safe area. Max 3 visible.
- `--vf-surface`, border, elevation 2, `rounded-lg`, `p-3`; status icon, title, optional action (Undo), and a close button.
- Success and info: `role="status"`, 5 s; pause on hover and focus. Errors: `role="alert"`, and they stay until dismissed. A toast with an action stays at least 8 s.
- Never the only record of an error in a form.

### 8.11 Tooltip

See [section 13](#13-tooltips-and-contextual-help). Style: `--vf-ink` background, `--vf-on-ink` text, `text-xs`, `rounded-md`, `px-2 py-1`, max width 240 px, `sideOffset={6}`, no arrow. Can hold a `Kbd` in the inverted style.

### 8.12 Command palette

Keep `CommandPalette.tsx` and `shortcuts/fuzzy.ts`.

- Dialog, width 640, top offset `15vh`, `rounded-xl`, elevation 3.
- Search input 48 px with no border, `Search` icon, `Esc` key cap.
- Results grouped (Navigation, Actions), items 40 px with icon, label and right-aligned `Shortcut`.
- Selected item is `--vf-surface-muted`, tracked by `aria-activedescendant`.
- Footer with hints: `↑↓` move, `Enter` run, `Esc` close. Hidden on touch.
- Empty result: "No results for {query}". Phones: full screen.

### 8.13 Amounts

`MoneyText` remains the only way to show money.

| Prop | Values |
| --- | --- |
| `kind` | `income`, `expense`, `transfer`, `neutral`, or derived from the sign |
| `size` | `sm`, `md`, `lg`, `hero` |
| `showIcon` | Direction icon, on by default in lists |
| `colored` | `true` in lists; `false` for totals where color adds noise |

Always: `tabular-nums`, sign before the currency symbol, U+2212 for minus, and the existing `aria-label` ("minus 12 dollars 50"). Currency and separators come from the locale. Hidden values, if ever added, are out of scope.

### 8.14 Skeleton

`ui/skeleton.tsx`: `--vf-surface-muted`, `rounded-md`, matching the final layout so nothing shifts. Show only after 200 ms of waiting. Pulse is an opacity animation between 1 and 0.6, opted in under `prefers-reduced-motion: no-preference`; otherwise static. The region sets `aria-busy="true"` and the live region announces "Loading" once.

### 8.15 Other components

| Component | Spec |
| --- | --- |
| `PageHeader` | One `h1` (focus target after navigation), description in `--vf-text-muted`, actions at the end; stacks on mobile; optional `HelpHint` |
| `Pagination` | Previous and next, page indicator, 36 px, keeps query |
| `Progress` (`LimitBar`, budgets, goals) | 8 px track `--vf-surface-muted`, fill `--vf-brand`; above 80% `--vf-warning`, above 100% `--vf-expense` with a stripe; always with the numbers in text |
| `DropdownMenu` (new, Radix) | Items 32 px, coarse 44, icon 16, shortcut on the end, destructive item in `--vf-expense` |
| `Popover` (new, Radix) | `rounded-lg`, elevation 2, `p-4`, max width 320 |
| `Avatar` | 28 px circle with initials, `--vf-surface-muted` |
| `Separator` | `--vf-border`, 1 px |
| `Kbd`, `Shortcut` | See [section 14](#14-keyboard-hints-on-buttons) |
| `Alert` (new) | Inline banner, status icon, title, text, optional action, subtle status background |
| `HelpHint`, `TermHint`, `HelpSheet` | See [section 13](#13-tooltips-and-contextual-help) |

## 9. Icons

**`lucide-react` is the only icon library** (ISC, already installed at 1.72.0). No upgrade is required. It already covers 32 files; no other icon package is present.

| Context | Size | Stroke |
| --- | --- | --- |
| Inline in `xs` and badges | 12 | 2 |
| `sm` button, table action | 14 | 2 |
| Default (buttons, nav, inputs) | 16 | 1.75 |
| `lg` button | 18 | 1.75 |
| Empty state | 24 | 1.5 |

Rules:

- Set the default once with CSS: `[data-slot] svg.lucide { stroke-width: 1.75; }`, overriding by size class only.
- Color is `currentColor`.
- Decorative icons get `aria-hidden="true"`. An icon-only control is named on the button.
- Import named icons individually, for tree-shaking.
- Meaning is fixed: `ArrowDownLeft` income, `ArrowUpRight` expense, `ArrowLeftRight` transfer, `Plus` create, `Pencil` edit, `Trash2` delete, `Search` search, `Download` export, `Upload` import, `CircleHelp` help, `Info` definition, `Keyboard` shortcuts, `MoreHorizontal` menu, `X` close.
- The current logo uses `ArrowUpRight`, which here means expense. T06 replaces it with a dedicated product mark as an inline SVG (see [open decision 4](#17-open-decisions)).
- The only non-Lucide glyphs allowed are third-party brand marks required by their guidelines, like the Google mark in the sign-in button.

## 10. Motion

Add to `tokens.css`:

```css
:root {
  --vf-duration-fast: 100ms; /* hover, color */
  --vf-duration-base: 150ms; /* popover, tooltip, menu */
  --vf-duration-slow: 200ms; /* dialog, sheet, tutorial step */
  --vf-ease-out: cubic-bezier(0.16, 1, 0.3, 1);
  --vf-ease-in-out: cubic-bezier(0.65, 0, 0.35, 1);
  --vf-shift: 4px; /* entry offset of floating layers */
  --vf-shift-sheet: 16px;
}

@media (prefers-reduced-motion: reduce) {
  :root {
    --vf-shift: 0px;
    --vf-shift-sheet: 0px;
  }
}
```

| Element | Motion | Duration |
| --- | --- | --- |
| Button, link, row | color only | fast |
| Tooltip, popover, menu | fade and `--vf-shift` from the anchor side | base |
| Dialog | fade and scale 0.98 to 1 | slow |
| Sheet | fade and `--vf-shift-sheet` | slow |
| Toast | fade and `--vf-shift` | slow |
| Tutorial step | fade, target outline appears | slow |
| Page change | none, or a 100 ms fade | fast |

Rules:

- Animate only `opacity` and `transform`, plus color for state.
- Exit is faster than entry (75% of its duration).
- No bounce, parallax, count-up numbers or animated chart entrances.
- **Reduced motion:** the change from today's rule is that durations no longer drop to zero. Spatial shift goes to zero through the tokens, scale is removed inside the same media query, and opacity and color transitions remain. Loops (spinner, skeleton) opt in with `prefers-reduced-motion: no-preference`; a spinner falls back to a static icon with the word "Loading".
- Never wait on `transitionend` without a fallback, because a transition may not run.
- Remove the `.finance-page` entrance animation.

## 11. Responsive rules

Tailwind breakpoints stay at their defaults.

| Range | Layout |
| --- | --- |
| below `sm` (under 640) | One column, gutter 16, sidebar in a sheet, header shows only trigger, title, search icon, add icon and avatar |
| `sm` to `md` (640 to 767) | One column, two-column stat grid |
| `md` to `lg` (768 to 1023) | Collapsed rail by default, gutter 24 |
| `lg` and up (1024+) | Full sidebar, gutter 32, multi-column layouts |
| `2xl` (1536+) | Content stays at 1200 and centered |

- The minimum supported width is 320 CSS px with no horizontal page scroll (WCAG 1.4.10).
- Prefer container queries (`@container`) for `Card`, `Stat` and charts, since their width depends on the sidebar.
- On phones, the primary page action moves to the header icon button or a bottom bar above the safe area: `padding-bottom: env(safe-area-inset-bottom)`.
- Use `dvh`, never `vh`, for full-height layers.
- Touch targets are at least 44 px on coarse pointers.
- Hover-only content does not exist: everything also works by tap and by keyboard.
- The desktop app (Wails) uses the same responsive layout.
- Print keeps `resources/css/print.css`: tables shown, shell hidden.

## 12. Accessibility

Target: **WCAG 2.2 AA**.

- **Focus.** Global `:focus-visible` outline, 2 px `--vf-focus`, 2 px offset. Never `outline: none` without a replacement. Inside tables and clipped containers use `outline-offset: -2px`. Replace `focus-visible:ring-*` utilities in primitives with this single ring. Focus is not hidden by sticky layers: set `scroll-padding-top: 72px` on the scroller.
- **Forced colors.** Keep the `Highlight` outline. States and boundaries use `border` or `outline`, never only shadows or backgrounds. Selected items get `forced-color-adjust: auto` with `Highlight`.
- **`prefers-contrast: more`.** `--vf-border` takes the value of `--vf-control-border`, and `--vf-text-muted` takes `--vf-text`.
- **Semantics.** One `h1` per page, no skipped levels, landmarks (`nav`, `main`, `header`, `search`), and native elements before ARIA.
- **Names.** Every control has a visible label or `aria-label`. Visible text is contained in the accessible name.
- **Forms.** Labels linked, `aria-invalid`, `aria-describedby`, required marked in text and with `required`, errors announced, and focus routed to the first error.
- **Live regions.** Keep the shell's polite region for navigation and async results.
- **Keyboard.** Everything operable; logical tab order; shortcuts follow `docs/shortcuts.md` and never fire while typing; single-key shortcuts can be turned off (WCAG 2.1.4).
- **Content on hover or focus (1.4.13).** Tooltips are dismissible with Esc, hoverable, and persistent.
- **Target size (2.5.8).** At least 24 px everywhere and 44 px on coarse pointers.
- **Dragging (2.5.7).** Any drag has a non-drag alternative.
- **Language.** `<html lang>` follows the locale, as `Preferences` does today.
- **Reflow and text.** Layouts survive 200% text and 400% zoom.
- **Testing.** Each screen task runs a keyboard pass, a 320 px pass, both themes, reduced motion, forced colors and an axe scan where the e2e harness allows it.

## 13. Tooltips and contextual help

### 13.1 Which pattern to use

| Pattern | Component | Trigger | Content | Use |
| --- | --- | --- | --- | --- |
| Tooltip | `Tooltip` (Radix) | Hover, focus | A few words | Name of an icon button, shortcut, why a control is disabled |
| Toggletip | `HelpHint` (Radix `Popover`) | A visible `CircleHelp` button: click, tap, Enter | 1 to 3 sentences, optional "Learn more" | How a number is calculated, what a setting does |
| Term | `TermHint` | Dotted underline button | Glossary definition | Financial terms in running text |
| Inline help | `FieldDescription` | Always visible | One line | Anything needed to complete a field |
| Page help | `HelpSheet` | `CircleHelp` in the header, `h p` | Why the screen exists, steps, shortcuts, tutorial button | Learning a screen |
| Empty state | `Empty` | Automatic | Why and first action | First use |

Information a person **needs to act** is never placed only in a tooltip.

### 13.2 Tooltip behavior

- `TooltipProvider delayDuration={400} skipDelayDuration={300}`.
- Never on disabled native buttons, because they do not receive events. Use `aria-disabled="true"` and block the action so the tooltip still works.
- If the tooltip only repeats the `aria-label`, add no description.
- Remove every native `title` used as a tooltip on interactive elements.
- The app stays on Radix positioning. Native `interestfor`, `popover="hint"` and CSS anchor positioning are not yet available in all engines (no Firefox anchor positioning, no Safari `hint`), so they are not used.

### 13.3 `HelpHint`

```tsx
interface HelpHintProps {
  /** Key in the `help` i18n domain, resolved to `<id>Title` and `<id>Body`. */
  id: HelpId;
  /** Accessible name, for example "About total balance". */
  labelKey: Key;
  side?: "top" | "right" | "bottom" | "left";
  /** Opens the page tutorial at this step. */
  tutorialStep?: string;
}
```

- Button: 16 px `CircleHelp` in `--vf-text-subtle`, hit area 24 px, 44 px on coarse.
- `aria-expanded` synchronized with the popover, and focus stays predictable: opening does not trap it, Esc closes and returns focus to the button.
- At most one `HelpHint` per field or `Stat`. If a screen needs more than about six, the screen needs a clearer design.

### 13.4 Content architecture

```text
resources/js/help/
  ids.ts          HelpId union
  HelpHint.tsx
  TermHint.tsx
  HelpSheet.tsx
  screens.ts      route to screen id, reused by the tutorial
resources/js/i18n/{en,pt-BR,es}/help.ts
```

Writing rules: say what it is, why it matters and how it is calculated, in that order. At most 3 short sentences. No jargon without a `TermHint`. Use the real product labels. Never put personal or financial data in help keys.

### 13.5 Glossary to ship in the `help` domain

| Key | en | pt-BR | es |
| --- | --- | --- | --- |
| `termBalance` | **Balance.** What an account holds: its opening balance plus every transaction. | **Saldo.** O que uma conta tem: o saldo inicial mais todas as transações. | **Saldo.** Lo que tiene una cuenta: su saldo inicial más todos los movimientos. |
| `termNetWorth` | **Net worth.** Everything you have minus everything you owe. | **Patrimônio líquido.** Tudo o que você tem menos tudo o que deve. | **Patrimonio neto.** Todo lo que tienes menos todo lo que debes. |
| `termCashFlow` | **Cash flow.** Money in minus money out over a period. | **Fluxo de caixa.** Dinheiro que entrou menos o que saiu em um período. | **Flujo de caja.** Dinero que entra menos el que sale en un periodo. |
| `termIncome` | **Income.** Money that arrives from outside your accounts. | **Receita.** Dinheiro que chega de fora das suas contas. | **Ingreso.** Dinero que llega desde fuera de tus cuentas. |
| `termExpense` | **Expense.** Money that leaves your accounts for good. | **Despesa.** Dinheiro que sai das suas contas de vez. | **Gasto.** Dinero que sale de tus cuentas definitivamente. |
| `termTransfer` | **Transfer.** Money moved between your own accounts. It is neither income nor expense. | **Transferência.** Dinheiro movido entre suas próprias contas. Não é receita nem despesa. | **Transferencia.** Dinero movido entre tus propias cuentas. No es ingreso ni gasto. |
| `termStatement` | **Statement.** The purchases of a card between two closing dates. | **Fatura.** As compras de um cartão entre duas datas de fechamento. | **Extracto.** Las compras de una tarjeta entre dos fechas de cierre. |
| `termClosingDay` | **Closing day.** The day a statement stops taking purchases. | **Dia de fechamento.** O dia em que a fatura para de receber compras. | **Día de cierre.** El día en que el extracto deja de recibir compras. |
| `termDueDay` | **Due day.** The last day to pay the statement. | **Dia de vencimento.** O último dia para pagar a fatura. | **Día de vencimiento.** El último día para pagar el extracto. |
| `termLimitUsed` | **Limit used.** How much of the card limit is taken right now. | **Limite usado.** Quanto do limite do cartão está comprometido agora. | **Límite usado.** Cuánto del límite de la tarjeta está ocupado ahora. |
| `termBudget` | **Budget.** A monthly limit you set for a category or for all spending. | **Orçamento.** Um limite mensal definido para uma categoria ou para todos os gastos. | **Presupuesto.** Un límite mensual para una categoría o para todos los gastos. |
| `termRollover` | **Carry over.** Adds what was left of last month's budget to this month. | **Levar a sobra.** Soma ao mês atual o que sobrou do orçamento do mês passado. | **Arrastrar.** Suma a este mes lo que sobró del presupuesto del mes pasado. |
| `termGoal` | **Goal.** An amount you want to reach, with an optional date. | **Meta.** Um valor que você quer alcançar, com data opcional. | **Meta.** Un importe que quieres alcanzar, con fecha opcional. |
| `termContribution` | **Contribution.** A note of progress on a goal. It does not move money. | **Contribuição.** Um registro de progresso em uma meta. Não move dinheiro. | **Aporte.** Una nota de progreso en una meta. No mueve dinero. |
| `termRecurrence` | **Recurring rule.** A rule that creates a transaction on each date it repeats. | **Regra recorrente.** Uma regra que cria uma transação em cada data em que se repete. | **Regla recurrente.** Una regla que crea un movimiento en cada fecha en que se repite. |
| `termInstallment` | **Installments.** One purchase split into monthly parts. | **Parcelas.** Uma compra dividida em partes mensais. | **Cuotas.** Una compra dividida en partes mensuales. |
| `termConfidence` | **Confidence.** How sure the suggestion is. Higher means more likely to be right. | **Confiança.** O quanto a sugestão é segura. Mais alta indica maior chance de acerto. | **Confianza.** Qué tan segura es la sugerencia. Más alta indica más probabilidad de acierto. |
| `termScope` | **Scope.** A permission of a token. Grant only what the tool needs. | **Escopo.** Uma permissão do token. Conceda só o que a ferramenta precisa. | **Permiso.** Un permiso del token. Concede solo lo que la herramienta necesita. |
| `termRole` | **Role.** What a person can do in this workspace: owner, admin, member or viewer. | **Papel.** O que uma pessoa pode fazer neste workspace: dono, admin, membro ou leitor. | **Rol.** Lo que una persona puede hacer en este espacio: propietario, admin, miembro o lector. |
| `termRecoveryCode` | **Recovery code.** A single-use code to sign in if you lose your authenticator. | **Código de recuperação.** Um código de uso único para entrar se perder o autenticador. | **Código de recuperación.** Un código de un solo uso para entrar si pierdes el autenticador. |

### 13.6 Where hints go

| Screen | `HelpHint` on |
| --- | --- |
| Dashboard | Total balance, Income, Expenses, Cash flow |
| Transactions | Type, Transfer, Tags |
| Accounts | Net worth, Opening balance, Archive |
| Cards | Limit used, Closing day, Due day |
| CardStatement | Total, Remaining, Pay from |
| Categories | Parent category, Reassign on delete |
| Budgets | Overall budget, Carry over |
| Goals | Contribution, Monthly amount |
| Recurrences | Frequency, Installments |
| Reports | Period, Net worth, Compare |
| Import | Date format, Duplicate, Positive amounts are expenses |
| Suggestions | Confidence, Source |
| AI | Provider key, Base URL, Monthly call limit |
| Workspace | Roles, Transfer ownership |
| Tokens | Scopes, Expiry |
| Devices | Last pull |
| Billing | Trial, PIX period |
| Notifications | Channels |
| TwoFactor | Recovery codes |

## 14. Keyboard hints on buttons

### 14.1 Rule

A button or menu item that triggers a registered action shows its shortcut. The registry is the only source: a key is **never** written by hand in JSX.

### 14.2 API

```tsx
// resources/js/shortcuts/useActionHint.ts
export interface ActionHint {
  /** One label per key press, for example ["G", "D"] or ["Ctrl+K"]. */
  parts: string[];
  /** Value for aria-keyshortcuts; undefined for a sequence. */
  ariaKeyShortcuts?: string;
}

export function useActionHint(actionId: string): ActionHint | null;
```

```tsx
// Button gets one prop:
<Button variant="brand" shortcutAction="new-expense">
  <Plus aria-hidden="true" />
  {t("addTransaction")}
</Button>
```

`Button` renders `ShortcutHint` at the end of its label, sets `aria-keyshortcuts` when available, and renders nothing extra when the hook returns `null`.

### 14.3 Behavior

`useActionHint` reads `useActions()`, finds the action and:

1. Returns `null` if the id is missing, has no `keys`, or `enabled()` is false.
2. Returns `null` for a binding without a modifier while the "Single-key shortcuts" preference is off (`useSingleKeyShortcuts`), because that shortcut would not work.
3. Formats with `formatBindingParts(keys, useIsMac())`. `useIsMac` is false on the server and first render, so the markup hydrates and then switches to the macOS symbols.

| Binding | Windows, Linux | macOS | `aria-keyshortcuts` |
| --- | --- | --- | --- |
| `mod+k` | `Ctrl+K` | `⌘+K` | `Control+K` or `Meta+K` |
| `?` | `?` | `?` | `Shift+?` |
| `[` | `[` | `[` | `[` |
| `/` | `/` | `/` | `/` |
| `x` | `X` | `X` | `X` |
| `g d` | `G` `D` | `G` `D` | omitted |

`aria-keyshortcuts` separates alternatives with spaces, so it **cannot** express a sequence. Sequences are omitted there and are listed in the shortcut help and the palette.

### 14.4 Visual

- `Kbd`: gains `data-slot="kbd"`, 18 px high, min width 18, `px-1`, `rounded-sm`, `kbd` type, `--vf-surface-muted`, 1 px `--vf-border`, `--vf-text-muted`.
- On `default` and `brand` buttons, `Kbd` uses a translucent variant: `bg-current/10`, `border-current/20`, `text-current`.
- Sequences show two caps with a 2 px gap. The tooltip writes them with a word: "G then D". The word is translated: `then`, `depois`, `luego`.
- The hint is `aria-hidden`; assistive tech gets `aria-keyshortcuts` or the shortcut help.

### 14.5 Hidden on touch

Visibility is CSS-only, so server and client markup match:

```css
[data-slot="shortcut-hint"] {
  display: none;
}

@media (hover: hover) and (pointer: fine) {
  [data-slot="shortcut-hint"] {
    display: inline-flex;
  }
}
```

The hint also hides below `sm` and in `xs` buttons. On an icon-only button it appears in the tooltip rather than the button.

### 14.6 Where hints appear

| Action id | Keys | Control |
| --- | --- | --- |
| `open-palette` | `mod+k` | Header search |
| `open-shortcut-help` | `?` | Help menu item |
| `toggle-sidebar` | `[` | Sidebar trigger (tooltip) |
| `go-*` | `g d`, `g t`, `g a`, `g k`, `g c`, `g b`, `g g`, `g u`, `g r`, `g i`, `g w` | Sidebar items, shown on hover or focus; tooltip in the rail |
| `review-suggestions` | `g s` | Suggestions sidebar item |
| `new-expense` | `n e` | Add transaction menu, New expense |
| `new-income` | `n i` | Add transaction menu, New income |
| `new-transfer` | `n t` | New transfer button and menu item |
| `search-transactions` | `/` | Transactions search field (placeholder hint) |
| `export-transactions` | `x` | Export menu |
| `ask-assistant` | `a s` | Assistant button |
| `open-tutorial` (new) | `h t` | Help menu, Start tutorial |
| `open-page-help` (new) | `h p` | Header help button |

The Suggestions nav item has no `keys` in `nav.ts`, while `review-suggestions` holds `g s`. Add an optional `actionId` to `NavItem`, so the sidebar reads the right action. Do not add `keys` to the item, because the registry would reject the duplicate.

The two new bindings are free: nothing uses the `h` prefix, and they carry no modifier, so `reserved.ts` accepts them. T04 adds them to the allocation table of `docs/shortcuts.md`.

## 15. The tutorial

### 15.1 Goals

Teach **what each screen is for and why it exists**, step by step, in the person's language, without blocking work and without touching their data.

### 15.2 Architecture

No dependency. Built on Radix `Popover` (anchoring, collision handling, focus, Esc) and the Inertia router.

```text
resources/js/tutorial/
  types.ts           TutorialScreen, TutorialStep, TutorialState
  screens.ts         the data of section 15.8, typed
  store.ts           load, save, migrate state
  useTutorial.ts     start, next, back, skipScreen, exit, resume, reset
  TutorialHost.tsx   mounted once in Shell
  TutorialStepCard.tsx
  TutorialCenter.tsx list of screens with progress, in HelpSheet
resources/js/i18n/{en,pt-BR,es}/tutorial.ts
```

```ts
export interface TutorialStep {
  /** Value of the data-tour attribute on the element to highlight. */
  target: string;
  titleKey: Key;
  bodyKey: Key;
  side?: "top" | "right" | "bottom" | "left";
}

export interface TutorialScreen {
  id: string;
  route: string;
  whyKey: Key;
  mode: "tour" | "guide";
  edition?: "saas" | "personal";
  /** Evaluated against shared props. Hidden when false. */
  visible?: (props: SharedProps) => boolean;
  steps: readonly TutorialStep[];
}

export interface TutorialState {
  version: 1;
  status: "idle" | "running" | "paused" | "done";
  screenId: string | null;
  stepIndex: number;
  completed: Record<string, true>;
  skipped: Record<string, true>;
  welcomeDismissed: boolean;
}
```

Highlight targets are stable attributes (`data-tour="dashboard-balance"`), never CSS classes or text.

### 15.3 Two modes

- **`tour`** (signed-in screens): a step card anchored to the target.
- **`guide`** (screens before sign-in, and `Error`): no spotlight, because there is no signed-in person or stored progress. A collapsed "How this works" `<details>` under the form shows the same steps.

### 15.4 Step card

- Non-modal: the page stays usable. Implemented as Radix `Popover` (`modal={false}`) with a virtual anchor, `onInteractOutside` prevented, and `role="dialog"` with `aria-labelledby` on its heading.
- Width 340 px; on phones a bottom sheet that does not cover the target (the page scrolls the target above it).
- Content: progress "Step 2 of 5" and a progress bar, `h3` title, body at 14/20, then **Back**, **Next** (`default` button), **Skip screen** and a close button. The last step of a screen reads **Next screen** and names it.
- The first step of each screen shows "Why this screen exists" with the `why` text.
- Each step's text makes sense on its own, without seeing the highlighted element.

Highlight:

- The target gets `data-tour-active`, styled with a 2 px `--vf-focus` outline and a 4 px offset. No dimming layer, no `box-shadow` cutout, so it survives forced colors and leaves the page operable.
- Scroll with `scrollIntoView({ block: "center" })`; use `behavior: "smooth"` only under `prefers-reduced-motion: no-preference`.
- Focus moves to the step heading (`tabindex="-1"`) only after the **person** starts or advances. On closing, focus returns to the target if focusable, or else to the Help button.

### 15.5 Start, resume, skip, reopen

| Event | Behavior |
| --- | --- |
| First signed-in visit | Dashboard shows a dismissible **Welcome card** inline: "Take the tour" and "Not now". The tour **never** opens by itself, and focus is not moved on load |
| Start | From the card, Help menu, `h t`, palette or an empty state's "Learn how" |
| Next on the last step | Goes to the next visible screen through `router.visit`, then resumes at step 1 once its `data-tour` targets exist |
| Back | Previous step, also across screens |
| Skip screen | Marks `skipped[id]`, goes on to the next screen |
| Close or Esc | `status: "paused"`, position kept. A toast says "Tutorial paused. Resume from Help" |
| Navigating away while running | Pauses; the Help button shows a small dot and "Resume tutorial" |
| Resume | From Help or the palette; opens the saved screen and step. If the step no longer exists, restarts that screen |
| Reopen one screen | `HelpSheet` of the screen, "Show me this screen", or `TutorialCenter` |
| Restart | `TutorialCenter`, "Restart tutorial", with confirmation. Clears progress only |
| Done | Final card with links to Help and shortcuts; `status: "done"` |

Missing target (empty data, role, plan, feature off):

1. Wait for the target up to 1 s using a `MutationObserver`, not polling.
2. If it is absent, show the same card centered without a highlight, with a note such as "This appears after you add your first account".
3. Never open a dialog, click, fill or submit on the person's behalf.

### 15.6 Persistence

- `localStorage` key: `vf.tutorial.v1.<userId>`, wrapped in `try/catch`, like `theme` and `vf.shortcuts.single`.
- Holds **only** screen ids, step index and flags; never amounts, names or page data. It stays on the device, so logging out does not need to clear it, and another user gets their own key.
- If storage fails, the tutorial still runs in memory for the session.
- A device change restarts progress. Server-side persistence is [open decision 5](#17-open-decisions).
- Increase `version` when steps change substantially; `store.ts` migrates or resets.

### 15.7 Rules

- **The tutorial never writes.** No financial or settings request is made by the tutorial.
- It respects edition, role, plan and feature flags: `personal` never shows Billing, Register or Revenue. Telegram appears only when the bot is configured. Admin screens appear only for panel admins.
- It uses no `keydown` listener. Its controls are buttons; the shortcuts are registry actions.
- While the card is open, the shortcut registry keeps working, because the card is not modal.
- Text comes from `i18n`; all three locales are required (`bun test resources/js/i18n` enforces it).
- Order: the sequence in the table below for signed-in screens; the first run covers the core path (Dashboard through Reports); other screens are offered as optional in `TutorialCenter`.

### 15.8 Screen inventory and content

There are **36 screens** (every `inertia.Define` in `app/pages/`, each reached from `routes/`) and **81 steps**. Dialogs belong to the screen that opens them.

| # | Screen id | Route | Page | Audience | Mode | Steps |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Home | `/` | `Home` | Visitors, `saas` only (`personal` redirects) | guide | 1 |
| 2 | Login | `/login` | `Auth/Login` | Anyone signed out | guide | 2 |
| 3 | Register | `/register` | `Auth/Register` | Visitors, `saas` only | guide | 1 |
| 4 | CheckEmail | `/register/check-email` | `Auth/CheckEmail` | New sign-ups | guide | 1 |
| 5 | VerifyEmail | `/email/verify` | `Auth/VerifyEmail` | Signed in, email not verified | guide | 1 |
| 6 | ForgotPassword | `/forgot-password` | `Auth/ForgotPassword` | Anyone signed out | guide | 1 |
| 7 | ResetPassword | `/reset-password/{token}` | `Auth/ResetPassword` | Holder of a reset link | guide | 1 |
| 8 | TwoFactorChallenge | `/two-factor-challenge` | `Auth/TwoFactorChallenge` | Account with two-factor on | guide | 1 |
| 9 | Setup | `/setup` | `Setup` | First owner, `personal` only | guide | 2 |
| 10 | InviteAccept | `/invites/{token}` | `InviteAccept` | Signed-in invitee | guide | 1 |
| 11 | InviteRegister | `/invites/{token}/register` | `InviteRegister` | Invitee without an account | guide | 1 |
| 12 | Error | `any failing route` | `Error` | Anyone | guide | 1 |
| 13 | Dashboard | `/dashboard` | `Dashboard` | Every member | tour | 5 |
| 14 | Transactions | `/transactions` | `Transactions` | Every member (writing needs a role that can record) | tour | 5 |
| 15 | Accounts | `/accounts` | `Accounts` | Every member | tour | 4 |
| 16 | Cards | `/cards` | `Cards` | Every member | tour | 3 |
| 17 | CardStatement | `/cards/{card}` | `CardStatement` | Every member | tour | 3 |
| 18 | Categories | `/categories` | `Categories` | Every member | tour | 3 |
| 19 | Budgets | `/budgets` | `Budgets` | Every member (Pro in `saas`) | tour | 3 |
| 20 | Goals | `/goals` | `Goals` | Every member | tour | 3 |
| 21 | Recurrences | `/recurrences` | `Recurrences` | Every member | tour | 3 |
| 22 | Reports | `/reports` | `Reports` | Every member | tour | 4 |
| 23 | Import | `/import` | `Import` | Members who can record (Pro in `saas`) | tour | 3 |
| 24 | Suggestions | `/suggestions` | `Suggestions` | Members who can record | tour | 3 |
| 25 | AI | `/ai` | `AI` | Owner manages; others read | tour | 3 |
| 26 | Workspace | `/workspace` | `Workspace` | Every member; owner and admin manage | tour | 3 |
| 27 | Notifications | `/notifications` | `Notifications` | Every member | tour | 2 |
| 28 | Telegram | `/settings/telegram` | `Settings/Telegram` | Every member, when the bot is configured | tour | 2 |
| 29 | Tokens | `/tokens` | `Tokens` | Members allowed to create tokens | tour | 3 |
| 30 | Devices | `/devices` | `Devices` | Every member | tour | 2 |
| 31 | Billing | `/billing` | `Billing` | Every member; owner pays, `saas` only | tour | 3 |
| 32 | TwoFactor | `/user/two-factor` | `Auth/TwoFactor` | Every signed-in person | tour | 2 |
| 33 | AdminDashboard | `/admin` | `Admin/Dashboard` | Panel administrators | tour | 2 |
| 34 | AdminResourceList | `/admin/{slug}` | `Admin/ResourceList` | Panel administrators | tour | 1 |
| 35 | AdminResourceShow | `/admin/{slug}/{id}` | `Admin/ResourceShow` | Panel administrators | tour | 1 |
| 36 | AdminRevenue | `/admin/revenue` | `Admin/Revenue` | Panel administrators, `saas` only | tour | 1 |

i18n keys are generated from the id and position: `tut<Id>Why`, `tut<Id><n>Title`, `tut<Id><n>Body`, in `resources/js/i18n/<locale>/tutorial.ts`.

Shared interface strings:

| Key | en | pt-BR | es |
| --- | --- | --- | --- |
| `tutorialStart` | Start tutorial | Iniciar tutorial | Iniciar tutorial |
| `tutorialResume` | Resume tutorial | Retomar tutorial | Reanudar tutorial |
| `tutorialRestart` | Restart tutorial | Reiniciar tutorial | Reiniciar tutorial |
| `tutorialRestartConfirm` | Restart from the beginning? Your data is not affected. | Recomeçar do início? Seus dados não são afetados. | ¿Empezar desde el principio? Tus datos no cambian. |
| `tutorialWelcomeTitle` | Welcome. Want a quick tour? | Boas-vindas. Quer um tour rápido? | Te damos la bienvenida. ¿Un recorrido rápido? |
| `tutorialWelcomeBody` | Learn what each screen is for. It changes nothing in your data, and you can stop at any time. | Aprenda para que serve cada tela. Nada muda nos seus dados e você pode parar quando quiser. | Aprende para qué sirve cada pantalla. No cambia tus datos y puedes parar cuando quieras. |
| `tutorialTakeTour` | Take the tour | Fazer o tour | Hacer el recorrido |
| `tutorialNotNow` | Not now | Agora não | Ahora no |
| `tutorialStepOf` | Step {current} of {total} | Passo {current} de {total} | Paso {current} de {total} |
| `tutorialWhy` | Why this screen exists | Por que esta tela existe | Por qué existe esta pantalla |
| `tutorialBack` | Back | Voltar | Atrás |
| `tutorialNext` | Next | Próximo | Siguiente |
| `tutorialNextScreen` | Next: {screen} | Próxima: {screen} | Siguiente: {screen} |
| `tutorialSkipScreen` | Skip screen | Pular tela | Saltar pantalla |
| `tutorialClose` | Close tutorial | Fechar tutorial | Cerrar tutorial |
| `tutorialPaused` | Tutorial paused. Resume it from Help. | Tutorial pausado. Retome pela Ajuda. | Tutorial en pausa. Reanúdalo desde Ayuda. |
| `tutorialMissingTarget` | This appears after you add your first item. | Isto aparece depois que você adicionar o primeiro item. | Esto aparece después de añadir tu primer elemento. |
| `tutorialShowScreen` | Show me this screen | Mostrar esta tela | Mostrar esta pantalla |
| `tutorialDoneTitle` | You are all set | Tudo pronto | Todo listo |
| `tutorialDoneBody` | Reopen any screen's guide from Help, or press ? to see every shortcut. | Reabra o guia de qualquer tela pela Ajuda ou pressione ? para ver todos os atalhos. | Reabre la guía de cualquier pantalla desde Ayuda o pulsa ? para ver todos los atajos. |
| `tutorialGuideSummary` | How this works | Como funciona | Cómo funciona |
| `tutorialThen` | then | depois | luego |

#### Home

Route `/`. Objective:

- **en:** Explains what the product does before anyone creates an account.
- **pt-BR:** Explica o que o produto faz antes de alguém criar uma conta.
- **es:** Explica qué hace el producto antes de crear una cuenta.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `home-cta` | **Start here.** Create a free account, or sign in if you already have one. | **Comece aqui.** Crie uma conta gratuita ou entre se já tiver uma. | **Empieza aquí.** Crea una cuenta gratuita o inicia sesión si ya tienes una. |

#### Login

Route `/login`. Objective:

- **en:** Confirms who you are, so only you can see your money.
- **pt-BR:** Confirma quem é você, para que só você veja o seu dinheiro.
- **es:** Confirma quién eres, para que solo tú veas tu dinero.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `login-form` | **Sign in.** Enter your email and password. Password managers work here. | **Entre.** Digite seu e-mail e sua senha. Gerenciadores de senha funcionam aqui. | **Inicia sesión.** Escribe tu correo y tu contraseña. Los gestores de contraseñas funcionan aquí. |
| 2 | `login-forgot` | **Forgot it?** Request a reset link. Your data stays untouched. | **Esqueceu?** Peça um link de redefinição. Seus dados continuam intactos. | **¿La olvidaste?** Pide un enlace de restablecimiento. Tus datos no cambian. |

#### Register

Route `/register`. Objective:

- **en:** Creates your account and your first workspace.
- **pt-BR:** Cria sua conta e seu primeiro workspace.
- **es:** Crea tu cuenta y tu primer espacio de trabajo.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `register-form` | **Create your account.** Use an email you can open: we send a confirmation link. | **Crie sua conta.** Use um e-mail que você consiga abrir: enviamos um link de confirmação. | **Crea tu cuenta.** Usa un correo al que tengas acceso: enviamos un enlace de confirmación. |

#### CheckEmail

Route `/register/check-email`. Objective:

- **en:** Tells you the next step happens in your inbox.
- **pt-BR:** Avisa que o próximo passo acontece na sua caixa de entrada.
- **es:** Indica que el siguiente paso ocurre en tu bandeja de entrada.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `check-email-message` | **Check your inbox.** Open the message we sent. Look in spam if it does not arrive. | **Veja seu e-mail.** Abra a mensagem que enviamos. Olhe o spam se ela não chegar. | **Revisa tu correo.** Abre el mensaje que enviamos. Mira en spam si no llega. |

#### VerifyEmail

Route `/email/verify`. Objective:

- **en:** Proves the address is yours before the app opens.
- **pt-BR:** Comprova que o endereço é seu antes de abrir o app.
- **es:** Comprueba que la dirección es tuya antes de abrir la app.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `verify-send-again` | **Confirm your email.** Use the link in the message, or ask for a new one here. | **Confirme seu e-mail.** Use o link da mensagem ou peça um novo aqui. | **Confirma tu correo.** Usa el enlace del mensaje o pide uno nuevo aquí. |

#### ForgotPassword

Route `/forgot-password`. Objective:

- **en:** Gets you back in when you lose the password.
- **pt-BR:** Devolve seu acesso quando você perde a senha.
- **es:** Te devuelve el acceso cuando pierdes la contraseña.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `forgot-form` | **Request a link.** Enter your email. If it has an account, a reset link is sent. | **Peça um link.** Digite seu e-mail. Se houver uma conta, enviamos um link de redefinição. | **Pide un enlace.** Escribe tu correo. Si tiene una cuenta, enviamos un enlace. |

#### ResetPassword

Route `/reset-password/{token}`. Objective:

- **en:** Sets a new password from a single-use link.
- **pt-BR:** Define uma nova senha a partir de um link de uso único.
- **es:** Define una contraseña nueva desde un enlace de un solo uso.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `reset-form` | **Choose a new password.** Pick one you do not use anywhere else, then sign in. | **Escolha uma nova senha.** Use uma que você não usa em outro lugar e depois entre. | **Elige una contraseña.** Usa una que no uses en otro sitio y luego inicia sesión. |

#### TwoFactorChallenge

Route `/two-factor-challenge`. Objective:

- **en:** Adds a second proof, so a stolen password is not enough.
- **pt-BR:** Adiciona uma segunda prova, para que uma senha roubada não baste.
- **es:** Añade una segunda prueba, para que una contraseña robada no baste.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `challenge-code` | **Enter your code.** Type the 6-digit code from your app, or use a recovery code. | **Digite seu código.** Digite o código de 6 dígitos do app ou use um código de recuperação. | **Escribe tu código.** Escribe el código de 6 dígitos de tu app o usa uno de recuperación. |

#### Setup

Route `/setup`. Objective:

- **en:** Creates the owner of a new installation, once.
- **pt-BR:** Cria o dono de uma nova instalação, uma única vez.
- **es:** Crea el propietario de una instalación nueva, una sola vez.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `setup-code` | **Setup code.** Find it in the server boot log, or in `SETUP_TOKEN`. | **Código de configuração.** Ele está no log de inicialização do servidor ou em `SETUP_TOKEN`. | **Código de configuración.** Está en el registro de arranque del servidor o en `SETUP_TOKEN`. |
| 2 | `setup-currency` | **Workspace currency.** Choose the currency your amounts are kept in. | **Moeda do workspace.** Escolha a moeda em que seus valores ficam guardados. | **Moneda del espacio.** Elige la moneda en la que se guardan tus importes. |

#### InviteAccept

Route `/invites/{token}`. Objective:

- **en:** Lets you join a workspace someone shared with you.
- **pt-BR:** Permite entrar em um workspace que alguém compartilhou com você.
- **es:** Te permite unirte a un espacio que alguien compartió contigo.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `invite-accept` | **Join the workspace.** Accepting gives you the role the inviter chose. | **Entre no workspace.** Ao aceitar, você recebe o papel escolhido por quem convidou. | **Únete al espacio.** Al aceptar recibes el rol que eligió quien te invitó. |

#### InviteRegister

Route `/invites/{token}/register`. Objective:

- **en:** Creates your account straight from an invitation.
- **pt-BR:** Cria sua conta direto a partir de um convite.
- **es:** Crea tu cuenta directamente desde una invitación.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `invite-register-form` | **Finish your account.** Add your name and a password. The email comes from the invitation. | **Conclua sua conta.** Informe seu nome e uma senha. O e-mail vem do convite. | **Completa tu cuenta.** Añade tu nombre y una contraseña. El correo viene de la invitación. |

#### Error

Route `any failing route`. Objective:

- **en:** Says what went wrong and offers a way back.
- **pt-BR:** Diz o que deu errado e oferece um caminho de volta.
- **es:** Dice qué salió mal y ofrece una forma de volver.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `error-back` | **Go back safely.** Your data was not changed by this page. Return and try again. | **Volte com segurança.** Esta página não alterou seus dados. Volte e tente de novo. | **Vuelve con seguridad.** Esta página no cambió tus datos. Vuelve e inténtalo de nuevo. |

#### Dashboard

Route `/dashboard`. Objective:

- **en:** Answers in seconds: how much do I have, and how is this month going?
- **pt-BR:** Responde em segundos: quanto eu tenho e como está este mês?
- **es:** Responde en segundos: ¿cuánto tengo y cómo va este mes?

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `dashboard-balance` | **Total balance.** The sum of your accounts. It is calculated from your transactions, never typed. | **Saldo total.** A soma das suas contas. É calculado pelas transações, nunca digitado. | **Saldo total.** La suma de tus cuentas. Se calcula con tus movimientos, nunca se escribe. |
| 2 | `dashboard-month` | **This month.** Income and expenses so far. Transfers between your accounts do not count. | **Este mês.** Receitas e despesas até agora. Transferências entre suas contas não contam. | **Este mes.** Ingresos y gastos hasta ahora. Las transferencias entre tus cuentas no cuentan. |
| 3 | `dashboard-cashflow` | **Cash flow.** Compare what came in and what went out, month by month. | **Fluxo de caixa.** Compare o que entrou e o que saiu, mês a mês. | **Flujo de caja.** Compara lo que entró y lo que salió, mes a mes. |
| 4 | `dashboard-upcoming` | **Coming up.** What is due in the next 30 days, so nothing surprises you. | **Próximos.** O que vence nos próximos 30 dias, para nada pegar você de surpresa. | **Próximos.** Lo que vence en los próximos 30 días, para que nada te sorprenda. |
| 5 | `quick-add` | **Add a transaction.** Available on every screen. Press N, then E for an expense. | **Adicione uma transação.** Disponível em todas as telas. Pressione N e depois E para uma despesa. | **Añade un movimiento.** Disponible en todas las pantallas. Pulsa N y luego E para un gasto. |

#### Transactions

Route `/transactions`. Objective:

- **en:** The full record of your money: every other screen is built from it.
- **pt-BR:** O registro completo do seu dinheiro: todas as outras telas nascem dele.
- **es:** El registro completo de tu dinero: todas las demás pantallas nacen de él.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `transactions-search` | **Search.** Find any entry by description. Press / to jump here. | **Busca.** Encontre qualquer lançamento pela descrição. Pressione / para vir aqui. | **Búsqueda.** Encuentra cualquier movimiento por su descripción. Pulsa / para venir aquí. |
| 2 | `transactions-filters` | **Filters.** Narrow by type, account, category, tag or date, then apply. | **Filtros.** Refine por tipo, conta, categoria, tag ou data e aplique. | **Filtros.** Acota por tipo, cuenta, categoría, etiqueta o fecha y aplica. |
| 3 | `transactions-table` | **Your entries.** Each amount shows a sign and a word, so it never depends on color. | **Seus lançamentos.** Cada valor mostra sinal e palavra, então nunca depende da cor. | **Tus movimientos.** Cada importe muestra signo y palabra, así nunca depende del color. |
| 4 | `transactions-transfer` | **Transfer.** Moving money between your accounts is not income or spending. | **Transferência.** Mover dinheiro entre suas contas não é receita nem gasto. | **Transferencia.** Mover dinero entre tus cuentas no es ingreso ni gasto. |
| 5 | `transactions-export` | **Export.** Download the filtered list as CSV. Press X. | **Exportar.** Baixe a lista filtrada em CSV. Pressione X. | **Exportar.** Descarga la lista filtrada en CSV. Pulsa X. |

#### Accounts

Route `/accounts`. Objective:

- **en:** Shows where your money is: bank, cash, savings or investments.
- **pt-BR:** Mostra onde seu dinheiro está: banco, dinheiro, poupança ou investimentos.
- **es:** Muestra dónde está tu dinero: banco, efectivo, ahorros o inversiones.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `accounts-networth` | **Net worth.** Everything you have minus what you owe, across all accounts. | **Patrimônio líquido.** Tudo o que você tem menos o que deve, em todas as contas. | **Patrimonio neto.** Todo lo que tienes menos lo que debes, en todas las cuentas. |
| 2 | `accounts-table` | **Balances.** A balance comes from the opening balance plus every transaction. | **Saldos.** O saldo vem do saldo inicial somado a todas as transações. | **Saldos.** El saldo viene del saldo inicial más todos los movimientos. |
| 3 | `accounts-new` | **New account.** Add one for each place that holds money. | **Nova conta.** Adicione uma para cada lugar que guarda dinheiro. | **Cuenta nueva.** Añade una por cada lugar que guarde dinero. |
| 4 | `accounts-archived` | **Archived accounts.** Archiving hides an account from pickers and keeps its history. | **Contas arquivadas.** Arquivar tira a conta das listas de escolha e mantém o histórico. | **Cuentas archivadas.** Archivar oculta la cuenta de los selectores y conserva su historial. |

#### Cards

Route `/cards`. Objective:

- **en:** Tracks credit card spending before the bill arrives.
- **pt-BR:** Acompanha os gastos no cartão antes de a fatura chegar.
- **es:** Sigue los gastos de la tarjeta antes de que llegue el extracto.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `cards-list` | **Your cards.** See the current statement, closing day and due day of each card. | **Seus cartões.** Veja a fatura atual, o fechamento e o vencimento de cada cartão. | **Tus tarjetas.** Mira el extracto actual, el cierre y el vencimiento de cada tarjeta. |
| 2 | `cards-limit` | **Limit used.** How much of the limit is taken right now. | **Limite usado.** Quanto do limite está comprometido agora. | **Límite usado.** Cuánto del límite está ocupado ahora. |
| 3 | `cards-new` | **New card.** We only store the brand and last 4 digits, never the full number. | **Novo cartão.** Guardamos só a bandeira e os 4 últimos dígitos, nunca o número completo. | **Tarjeta nueva.** Solo guardamos la marca y los 4 últimos dígitos, nunca el número completo. |

#### CardStatement

Route `/cards/{card}`. Objective:

- **en:** Shows what one statement contains and lets you pay it.
- **pt-BR:** Mostra o que uma fatura contém e permite pagá-la.
- **es:** Muestra qué contiene un extracto y permite pagarlo.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `statement-picker` | **Statements.** Move between past, open and future statements. | **Faturas.** Navegue entre faturas passadas, aberta e futuras. | **Extractos.** Muévete entre extractos pasados, abierto y futuros. |
| 2 | `statement-totals` | **Totals.** Purchases, refunds, what was paid and what remains. | **Totais.** Compras, estornos, o que foi pago e o que falta. | **Totales.** Compras, devoluciones, lo pagado y lo que falta. |
| 3 | `statement-pay` | **Pay statement.** Paying is a transfer from an account, so the purchases are not counted twice. | **Pagar fatura.** Pagar é uma transferência de uma conta, então as compras não contam duas vezes. | **Pagar extracto.** Pagar es una transferencia desde una cuenta, así las compras no cuentan dos veces. |

#### Categories

Route `/categories`. Objective:

- **en:** Gives each transaction a meaning, so reports and budgets can add it up.
- **pt-BR:** Dá significado a cada transação, para relatórios e orçamentos somarem.
- **es:** Da sentido a cada movimiento, para que informes y presupuestos lo sumen.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `categories-kind` | **Expense or income.** Each category belongs to one kind. | **Despesa ou receita.** Cada categoria pertence a um tipo. | **Gasto o ingreso.** Cada categoría pertenece a un tipo. |
| 2 | `categories-list` | **Groups.** Nest one level, such as Home and then Rent. | **Grupos.** Aninhe um nível, como Casa e depois Aluguel. | **Grupos.** Anida un nivel, como Hogar y luego Alquiler. |
| 3 | `categories-tags` | **Tags.** Tags cross categories: use one for a trip or a project. | **Tags.** Tags cruzam categorias: use uma para uma viagem ou um projeto. | **Etiquetas.** Las etiquetas cruzan categorías: usa una para un viaje o un proyecto. |

#### Budgets

Route `/budgets`. Objective:

- **en:** Sets a monthly limit, so you decide before you spend.
- **pt-BR:** Define um limite mensal, para você decidir antes de gastar.
- **es:** Fija un límite mensual, para decidir antes de gastar.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `budgets-month` | **Month.** Budgets are monthly. Move between months here. | **Mês.** Orçamentos são mensais. Navegue entre os meses aqui. | **Mes.** Los presupuestos son mensuales. Cambia de mes aquí. |
| 2 | `budgets-progress` | **Progress.** Each bar shows spent and limit in numbers as well. | **Progresso.** Cada barra mostra também o gasto e o limite em números. | **Progreso.** Cada barra muestra también lo gastado y el límite en números. |
| 3 | `budgets-form` | **Set a budget.** Choose a category or the overall budget, and optionally carry over what is left. | **Defina um orçamento.** Escolha uma categoria ou o orçamento geral e, se quiser, leve a sobra adiante. | **Fija un presupuesto.** Elige una categoría o el presupuesto general y, si quieres, arrastra lo que sobre. |

#### Goals

Route `/goals`. Objective:

- **en:** Turns a wish into a monthly amount you can follow.
- **pt-BR:** Transforma um desejo em um valor mensal que dá para seguir.
- **es:** Convierte un deseo en una cantidad mensual que puedes seguir.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `goals-new` | **New goal.** Name it, set a target and, if you want, a date. | **Nova meta.** Dê um nome, defina um valor e, se quiser, uma data. | **Meta nueva.** Ponle nombre, fija un importe y, si quieres, una fecha. |
| 2 | `goals-progress` | **Progress.** See how much is left and how much to set aside each month. | **Progresso.** Veja quanto falta e quanto guardar por mês. | **Progreso.** Mira cuánto falta y cuánto apartar cada mes. |
| 3 | `goals-contribution` | **Contributions.** A contribution is a note of progress. It does not move money between accounts. | **Contribuições.** A contribuição é um registro de progresso. Ela não move dinheiro entre contas. | **Aportes.** Un aporte es una nota de progreso. No mueve dinero entre cuentas. |

#### Recurrences

Route `/recurrences`. Objective:

- **en:** Records repeating bills and income for you, so nothing is forgotten.
- **pt-BR:** Registra contas e receitas repetidas por você, para nada ser esquecido.
- **es:** Registra por ti facturas e ingresos repetidos, para no olvidar nada.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `recurrences-upcoming` | **Coming up.** The dates generated in the next 30 days. | **Próximos.** As datas geradas nos próximos 30 dias. | **Próximos.** Las fechas generadas en los próximos 30 días. |
| 2 | `recurrences-table` | **Your rules.** Each rule shows frequency, next dates and status. Pause it at any time. | **Suas regras.** Cada regra mostra frequência, próximas datas e status. Pause quando quiser. | **Tus reglas.** Cada regla muestra frecuencia, próximas fechas y estado. Páusala cuando quieras. |
| 3 | `recurrences-new` | **New rule.** Create it once, and transactions appear on their dates. | **Nova regra.** Crie uma vez e as transações aparecem nas datas certas. | **Regla nueva.** Créala una vez y los movimientos aparecen en sus fechas. |

#### Reports

Route `/reports`. Objective:

- **en:** Shows patterns over time that one month cannot show.
- **pt-BR:** Mostra padrões ao longo do tempo que um único mês não mostra.
- **es:** Muestra patrones en el tiempo que un solo mes no muestra.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `reports-period` | **Period.** Choose a preset or custom dates, then apply. | **Período.** Escolha um período pronto ou datas próprias e aplique. | **Periodo.** Elige un periodo predefinido o fechas propias y aplica. |
| 2 | `reports-tabs` | **Views.** Cash flow, categories, accounts, net worth and comparison. | **Visões.** Fluxo de caixa, categorias, contas, patrimônio e comparação. | **Vistas.** Flujo de caja, categorías, cuentas, patrimonio y comparación. |
| 3 | `reports-table-toggle` | **Show as table.** Every chart has the same data as a table. | **Ver como tabela.** Todo gráfico tem os mesmos dados em tabela. | **Ver como tabla.** Cada gráfico tiene los mismos datos en una tabla. |
| 4 | `reports-export` | **Export.** Take the report with you as a file. Press X. | **Exportar.** Leve o relatório em um arquivo. Pressione X. | **Exportar.** Llévate el informe en un archivo. Pulsa X. |

#### Import

Route `/import`. Objective:

- **en:** Brings your bank history in, so you do not type it all.
- **pt-BR:** Traz seu histórico do banco, para você não digitar tudo.
- **es:** Trae tu historial del banco, para que no lo escribas todo.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `import-upload` | **Upload a statement.** Choose the account and a CSV or OFX file up to 2 MiB. | **Envie um extrato.** Escolha a conta e um arquivo CSV ou OFX de até 2 MiB. | **Sube un extracto.** Elige la cuenta y un archivo CSV u OFX de hasta 2 MiB. |
| 2 | `import-mapping` | **Match the columns.** Tell us which column is the date, the description and the amount. | **Associe as colunas.** Diga qual coluna é a data, a descrição e o valor. | **Asocia las columnas.** Indica qué columna es la fecha, la descripción y el importe. |
| 3 | `import-preview` | **Review first.** Nothing is saved yet. Duplicates and unreadable rows are marked. | **Revise antes.** Nada foi salvo ainda. Duplicadas e linhas ilegíveis ficam marcadas. | **Revisa primero.** Aún no se guarda nada. Se marcan los duplicados y las filas ilegibles. |

#### Suggestions

Route `/suggestions`. Objective:

- **en:** Clears uncategorized entries quickly, and you approve each change.
- **pt-BR:** Limpa os lançamentos sem categoria rapidamente, com sua aprovação.
- **es:** Despeja rápido los movimientos sin categoría, con tu aprobación.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `suggestions-list` | **Review.** Each row says if it came from your history or from AI, and its confidence. | **Revise.** Cada linha diz se veio do seu histórico ou da IA, e a confiança. | **Revisa.** Cada fila indica si viene de tu historial o de la IA, y su confianza. |
| 2 | `suggestions-actions` | **Accept or reject.** Use the arrow keys, Enter to accept and Backspace to reject. | **Aceite ou rejeite.** Use as setas, Enter para aceitar e Backspace para rejeitar. | **Acepta o rechaza.** Usa las flechas, Enter para aceptar y Retroceso para rechazar. |
| 3 | `suggestions-bulk` | **Accept in bulk.** Accept all suggestions at or above the confidence you choose. | **Aceite em lote.** Aceite todas as sugestões com a confiança mínima escolhida. | **Acepta en lote.** Acepta todas las sugerencias con la confianza mínima elegida. |

#### AI

Route `/ai`. Objective:

- **en:** Lets the owner turn on optional AI with their own provider key.
- **pt-BR:** Permite ao dono ligar a IA opcional com a própria chave do provedor.
- **es:** Permite al propietario activar la IA opcional con su propia clave.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `ai-status` | **Off by default.** AI runs only after the owner turns it on. | **Desligada por padrão.** A IA só funciona depois que o dono a liga. | **Apagada por defecto.** La IA solo funciona cuando el propietario la activa. |
| 2 | `ai-key` | **Provider key.** Your key is stored encrypted and is never shown again. | **Chave do provedor.** Sua chave fica criptografada e nunca mais é exibida. | **Clave del proveedor.** Tu clave se guarda cifrada y no se vuelve a mostrar. |
| 3 | `ai-limits` | **Switches and limit.** Choose the features and a monthly call limit. | **Opções e limite.** Escolha os recursos e um limite mensal de chamadas. | **Opciones y límite.** Elige las funciones y un límite mensual de llamadas. |

#### Workspace

Route `/workspace`. Objective:

- **en:** Controls who shares this money and what each person can do.
- **pt-BR:** Controla quem compartilha este dinheiro e o que cada pessoa pode fazer.
- **es:** Controla quién comparte este dinero y qué puede hacer cada persona.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `workspace-members` | **Members and roles.** Owner, admin, member and viewer have different permissions. | **Membros e papéis.** Dono, admin, membro e leitor têm permissões diferentes. | **Miembros y roles.** Propietario, admin, miembro y lector tienen permisos distintos. |
| 2 | `workspace-invite` | **Invite someone.** Send an invitation and choose the role. | **Convide alguém.** Envie um convite e escolha o papel. | **Invita a alguien.** Envía una invitación y elige el rol. |
| 3 | `workspace-export` | **Your data.** Owners and admins can export the whole workspace as JSON. | **Seus dados.** Donos e admins podem exportar todo o workspace em JSON. | **Tus datos.** Propietarios y admins pueden exportar todo el espacio en JSON. |

#### Notifications

Route `/notifications`. Objective:

- **en:** Collects the alerts that need your attention in one place.
- **pt-BR:** Reúne em um só lugar os alertas que precisam da sua atenção.
- **es:** Reúne en un solo lugar las alertas que necesitan tu atención.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `notifications-list` | **Alerts.** Unread items are marked with a word as well as a dot. | **Alertas.** Itens não lidos têm uma palavra além do ponto. | **Alertas.** Los elementos no leídos llevan una palabra además del punto. |
| 2 | `notifications-preferences` | **Preferences.** Choose which alerts reach the app and your email. | **Preferências.** Escolha quais alertas chegam ao app e ao seu e-mail. | **Preferencias.** Elige qué alertas llegan a la app y a tu correo. |

#### Telegram

Route `/settings/telegram`. Objective:

- **en:** Lets you record and check money from a chat.
- **pt-BR:** Permite registrar e consultar seu dinheiro por um chat.
- **es:** Permite registrar y consultar tu dinero desde un chat.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `telegram-connect` | **Connect a chat.** Create a single-use link that expires in minutes. | **Conecte um chat.** Crie um link de uso único que expira em minutos. | **Conecta un chat.** Crea un enlace de un solo uso que caduca en minutos. |
| 2 | `telegram-chats` | **Linked chats.** See what is connected and disconnect at any time. | **Chats vinculados.** Veja o que está conectado e desconecte quando quiser. | **Chats vinculados.** Mira qué está conectado y desconecta cuando quieras. |

#### Tokens

Route `/tokens`. Objective:

- **en:** Gives scripts and tools limited access without your password.
- **pt-BR:** Dá a scripts e ferramentas acesso limitado sem usar sua senha.
- **es:** Da a scripts y herramientas acceso limitado sin usar tu contraseña.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `tokens-new` | **New token.** Grant only the scopes the tool needs, and an expiry. | **Novo token.** Conceda só os escopos necessários e uma validade. | **Token nuevo.** Concede solo los permisos necesarios y una caducidad. |
| 2 | `tokens-copy` | **Copy it now.** The token is shown once. Keep it in a safe place. | **Copie agora.** O token aparece uma única vez. Guarde em local seguro. | **Cópialo ahora.** El token se muestra una vez. Guárdalo en un lugar seguro. |
| 3 | `tokens-table` | **Revoke.** Revoke a token as soon as it is no longer needed. | **Revogue.** Revogue um token assim que ele não for mais necessário. | **Revoca.** Revoca un token en cuanto deje de hacer falta. |

#### Devices

Route `/devices`. Objective:

- **en:** Shows which devices sync this workspace, and lets you cut one off.
- **pt-BR:** Mostra quais dispositivos sincronizam este workspace e permite bloquear um.
- **es:** Muestra qué dispositivos sincronizan este espacio y permite cortar uno.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `devices-table` | **Your devices.** Check the platform and when each device was last seen. | **Seus dispositivos.** Confira a plataforma e quando cada um foi visto pela última vez. | **Tus dispositivos.** Revisa la plataforma y cuándo se vio cada uno por última vez. |
| 2 | `devices-revoke` | **Lost a device?** Revoke it here to stop its sync. | **Perdeu um dispositivo?** Revogue aqui para interromper a sincronização. | **¿Perdiste un dispositivo?** Revócalo aquí para detener su sincronización. |

#### Billing

Route `/billing`. Objective:

- **en:** Shows your plan and how to change it.
- **pt-BR:** Mostra seu plano e como alterá-lo.
- **es:** Muestra tu plan y cómo cambiarlo.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `billing-plan` | **Your plan.** See the current state and the limits of Free and Pro. | **Seu plano.** Veja a situação atual e os limites de Free e Pro. | **Tu plan.** Mira el estado actual y los límites de Free y Pro. |
| 2 | `billing-card` | **Pay by card.** A subscription that renews until you cancel. | **Pagar com cartão.** Uma assinatura que renova até você cancelar. | **Pagar con tarjeta.** Una suscripción que se renueva hasta que la canceles. |
| 3 | `billing-pix` | **Pay with PIX.** A one-off payment for the months you choose, with no renewal. | **Pagar com PIX.** Um pagamento único pelos meses escolhidos, sem renovação. | **Pagar con PIX.** Un pago único por los meses que elijas, sin renovación. |

#### TwoFactor

Route `/user/two-factor`. Objective:

- **en:** Protects the account even if the password leaks.
- **pt-BR:** Protege a conta mesmo se a senha vazar.
- **es:** Protege la cuenta aunque se filtre la contraseña.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `twofactor-enable` | **Turn it on.** Scan the QR code with an authenticator app. | **Ative.** Leia o QR code com um app autenticador. | **Actívala.** Escanea el código QR con una app de autenticación. |
| 2 | `twofactor-recovery` | **Recovery codes.** Save them: they are your way in if you lose the phone. | **Códigos de recuperação.** Guarde-os: são seu acesso se perder o celular. | **Códigos de recuperación.** Guárdalos: son tu acceso si pierdes el teléfono. |

#### AdminDashboard

Route `/admin`. Objective:

- **en:** Gives administrators an overview of the installation.
- **pt-BR:** Dá aos administradores uma visão geral da instalação.
- **es:** Da a los administradores una visión general de la instalación.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `admin-overview` | **Overview.** Totals of the resources in this installation. | **Visão geral.** Totais dos recursos desta instalação. | **Visión general.** Totales de los recursos de esta instalación. |
| 2 | `admin-nav` | **Resources.** Open a resource from the menu to inspect its records. | **Recursos.** Abra um recurso pelo menu para inspecionar os registros. | **Recursos.** Abre un recurso desde el menú para inspeccionar sus registros. |

#### AdminResourceList

Route `/admin/{slug}`. Objective:

- **en:** Finds records of one resource for support and operation.
- **pt-BR:** Encontra registros de um recurso para suporte e operação.
- **es:** Encuentra registros de un recurso para soporte y operación.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `admin-list-table` | **Records.** Search, sort and page through the list, then open a row. | **Registros.** Busque, ordene e pagine a lista e abra uma linha. | **Registros.** Busca, ordena y pagina la lista, y abre una fila. |

#### AdminResourceShow

Route `/admin/{slug}/{id}`. Objective:

- **en:** Shows every field of one record.
- **pt-BR:** Mostra todos os campos de um registro.
- **es:** Muestra todos los campos de un registro.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `admin-show-fields` | **Record details.** Read the fields of this record. Go back to the list when done. | **Detalhes do registro.** Leia os campos deste registro. Volte à lista ao terminar. | **Detalles del registro.** Lee los campos de este registro. Vuelve a la lista al terminar. |

#### AdminRevenue

Route `/admin/revenue`. Objective:

- **en:** Tracks what the service earns.
- **pt-BR:** Acompanha o que o serviço fatura.
- **es:** Sigue lo que factura el servicio.

| # | `data-tour` (highlight) | en | pt-BR | es |
| --- | --- | --- | --- | --- |
| 1 | `admin-revenue-summary` | **Revenue.** Read recurring revenue, subscription states and recent PIX payments. | **Receita.** Veja a receita recorrente, a situação das assinaturas e os PIX recentes. | **Ingresos.** Consulta los ingresos recurrentes, el estado de las suscripciones y los PIX recientes. |

## 16. Migration plan

### 16.1 How to run it

- Each task is one branch and one PR or a small commit series, and leaves the app working. Scopes: `ui` unless noted; see `.github/scopes.txt`.
- **Order:** T01 to T07 run **in sequence**, because each one changes shared files. T08 to T16 can then run **in parallel**: each owns its files. T17 closes.
- **File ownership prevents collisions.** A task edits only the files listed. If a screen task needs a change in a shared component, it asks the owner or files a follow-up. It does not edit it.
- `i18n/*/common.ts`, `components.ts`, `shell.ts` and `keys.ts` are edited only in T01 to T07. Screen tasks use their own domain file.
- New `data-tour` attributes, `HelpHint`s and `shortcutAction` props are added by the task that owns the screen.
- e2e specs are updated by the task that owns the behavior. They should query roles and names, so a visual change should not break them.

Compatibility: T01 keeps every existing `--vf-*` and shadcn name, so pages not yet migrated keep rendering. T02 keeps the current variant and size names and only adds new ones.

### 16.2 Common acceptance criteria

Every task must, before it is done:

- Pass `bun run typecheck`, `bun run lint:js`, `bun run format:check`, `bun run test:js`, `bun run lint:md` and `bun run build`, plus the affected `bunx playwright test e2e/<spec>`.
- Show the affected screens correctly in light and dark at 320, 768 and 1440 px.
- Keep keyboard operation with visible focus; work under reduced motion and forced colors.
- Add no hardcoded color (`zinc-*`, hex, `text-white`) or string; all text in `en`, `pt-BR` and `es`.
- Use only Lucide icons; no native `title` tooltips on the screens it owns.
- Change no route, prop, permission or money logic.

### 16.3 Foundations (sequential)

#### T01. Tokens, fonts, theme base

- **Files:** `resources/css/tokens.css`, `finance.css`, `app.css`, `resources/views/app.html`, `resources/js/app.tsx` (font imports only), `resources/js/design/contrast.test.ts`, `package.json`, `bun.lock`, `docs/design.md`, `AGENTS.md`.
- **Work:** sections 4, 5, 6 and 10. Add the two font packages at the exact versions. Keep the `finance-landing` classes until T16. Point `docs/design.md` to this document and update the design paragraph of `AGENTS.md`.
- **Screens that change:** all (color, type and spacing tokens only).
- **Accept:** computed `font-family` is Geist; the contrast test covers every pair of 4.4 and passes; no flash of the wrong theme; System follows the OS live.

#### T02. Primitives

- **Files:** `components/ui/*`, new `ui/select.tsx`, `ui/popover.tsx`, `ui/dropdown-menu.tsx`, `ui/checkbox.tsx`, `ui/switch.tsx`, `ui/progress.tsx`, `ui/alert.tsx`, `ui/avatar.tsx`, and `components.json`.
- **Work:** 8.1 to 8.11 for the primitive layer. `sidebar.tsx` only gets token changes.
- **Screens that change:** all (controls).
- **Accept:** every variant, size and state exists; no `transition-all`; 44 px on a coarse pointer; existing call sites compile unchanged.

#### T03. Shared domain components

- **Files:** `components/{MoneyText,AmountInput,DataTable,PageHeader,Pagination,Toast,Combobox,ConfirmDelete,Field,PasswordField,ExportMenu}.tsx`, `components/ui.tsx`, `components/charts/*`, new `components/Stat.tsx`, `i18n/*/components.ts`.
- **Work:** 4.6, 4.7, 8.2, 8.4, 8.10, 8.13, 8.15.
- **Screens that change:** Dashboard, Transactions, Accounts, Cards, CardStatement, Budgets, Goals, Recurrences, Reports, Import, Tokens, Devices.
- **Accept:** income and expense are distinguishable in grayscale; table works as a list below `md`; charts keep their table view.

#### T04. Keyboard hints

- **Files:** `shortcuts/useActionHint.ts` (new), `shortcuts/format.ts`, `components/Kbd.tsx`, `components/ui/button.tsx` (the `shortcutAction` prop), `shortcuts/builtin.ts`, `shortcuts/shortcuts.test.ts`, `docs/shortcuts.md`, `i18n/*/shell.ts`.
- **Work:** section 14, the `open-tutorial` and `open-page-help` specs, and `actionId` on `NavItem` in `nav.ts`.
- **Screens that change:** none visibly until adopted, apart from the palette and shortcut help.
- **Accept:** unit tests cover Windows and macOS, a sequence, disabled actions and single keys off; hint hidden under `hover: none`; `builtin.test.ts` still matches `docs/shortcuts.md`.

#### T05. Contextual help

- **Files:** `resources/js/help/*` (new), `i18n/*/help.ts` (new), `i18n/keys.ts`, `components/ui/tooltip.tsx`.
- **Work:** section 13, with the glossary.
- **Screens that change:** none until adopted.
- **Accept:** `HelpHint` works by mouse, touch, keyboard and Esc; screen reader names are correct; the i18n test passes in all locales.

#### T06. Shell and auth layout

- **Files:** `layouts/Shell.tsx`, `layouts/AppLayout.tsx`, new `layouts/AuthLayout.tsx`, `nav.ts`, `components/{Preferences,CommandPalette,ShortcutHelp,BellButton,NotificationBell}.tsx`, `i18n/*/shell.ts`, `e2e/shell.spec.ts`, `e2e/palette.spec.ts`, `e2e/help.spec.ts`.
- **Work:** 7.1 frame, 7.2, 4.5 control, 8.12, the product mark, the mount point for `TutorialHost`.
- **Screens that change:** every signed-in screen (frame) and every auth screen (frame).
- **Accept:** sidebar groups by edition; collapsed rail has tooltips with shortcuts; `personal` shows no billing or workspace switcher; skip link and focus-after-navigation still work.

#### T07. Tutorial engine and copy

- **Files:** `resources/js/tutorial/*` (new), `i18n/*/tutorial.ts` (new), `i18n/keys.ts`, one mount line in `layouts/Shell.tsx`, new `e2e/tutorial.spec.ts`.
- **Work:** section 15, all copy. A missing target falls back to a centered card, so the engine can ship before the screens get their `data-tour` attributes.
- **Screens that change:** Dashboard (Welcome card), Help menu.
- **Accept:** start, next, back, skip, pause, resume, reopen and restart are covered by e2e; it makes no write request (asserted on the network); it respects edition and role; progress survives a reload.

### 16.4 Screens (parallel after T07)

| Task | Owned files | Screens that change | Specific acceptance |
| --- | --- | --- | --- |
| **T08** Auth | `pages/Auth/{Login,Register,ForgotPassword,ResetPassword,CheckEmail,VerifyEmail,TwoFactorChallenge}.tsx`, `pages/{Setup,InviteAccept,InviteRegister,Error}.tsx` | Login, Register, CheckEmail, VerifyEmail, ForgotPassword, ResetPassword, TwoFactorChallenge, Setup, InviteAccept, InviteRegister, Error | Layout 7.1; correct `autocomplete`; errors focus the first field; `personal` has no sign-up link |
| **T09** Overview | `pages/Dashboard.tsx`, `pages/Reports.tsx`, `i18n/*/reports.ts`, `e2e/reports.spec.ts` | Dashboard, Reports | Layout 7.6; hero number first; each chart has a table; hints on the figures |
| **T10** Transactions | `pages/Transactions.tsx`, `components/transactions/*`, `i18n/*/{transactions,search}.ts`, `e2e/{search,transactions-quick-add}.spec.ts` | Transactions, quick add, transfer dialog | `/` and `n` shortcuts unchanged; filters become a sheet on mobile |
| **T11** Accounts and cards | `pages/{Accounts,Cards,CardStatement}.tsx`, `components/cards/*`, `i18n/*/{accounts,cards}.ts`, `e2e/cards.spec.ts` | Accounts, Cards, CardStatement | Limit shown in numbers; statement status has word and icon |
| **T12** Planning | `pages/{Categories,Budgets,Goals}.tsx`, `components/categories/*`, `i18n/*/{categories,budgets,goals}.ts`, `e2e/import-categories.spec.ts` (categories part) | Categories, Budgets, Goals | Progress has text; category swatch never the only signal |
| **T13** Automation | `pages/{Recurrences,Import,Suggestions}.tsx`, `components/recurrences/*`, `i18n/*/{recurrences,import}.ts`, `e2e/{recurrences,ai-suggestions}.spec.ts`, `e2e/import-categories.spec.ts` (import part, after T12) | Recurrences, Import, Suggestions | Import shows numbered steps; local keys of Suggestions unchanged |
| **T14** Workspace and billing | `pages/{Workspace,Billing,Notifications}.tsx`, `pages/Settings/Telegram.tsx`, `components/PixDialog.tsx`, `i18n/*/{billing,alerts,telegram}.ts`, `e2e/{billing,alerts,telegram-settings}.spec.ts` | Workspace, Billing, Notifications, Telegram | Layout 7.7; owner-only controls explained when disabled |
| **T15** Security and AI | `pages/{AI,Tokens,Devices}.tsx`, `pages/Auth/TwoFactor.tsx`, `components/assistant/*`, `i18n/*/{ai,tokens,sync}.ts`, `e2e/{ai-settings,assistant,tokens,devices}.spec.ts` | AI, Tokens, Devices, TwoFactor, assistant | Secrets are shown once with Copy; TwoFactor has no hardcoded English |
| **T16** Public and admin | `pages/Home.tsx`, `pages/Admin/*`, `layouts/AdminLayout.tsx`, the `finance-landing` rules of `finance.css` | Home, Admin Dashboard, ResourceList, ResourceShow, Revenue | Admin has no `zinc-*`; landing composition is original |

### 16.5 Close

#### T17. Audit and cleanup

- **Files:** any leftover from the search below, `docs/design.md`, `AGENTS.md`.
- **Work:** remove aliases and dead CSS; run the searches below; review every screen in both themes.
- **Screens that change:** only those with leftovers.
- **Accept:** these return nothing in `resources/js` (tests excluded): `zinc-`, `transition-all`, `text-white`, `bg-white`, `title={t(` on a control, an import of an icon package other than `lucide-react`; every route of 15.8 has its tutorial; a full `bunx playwright test` passes.

### 16.6 Dependency graph

```text
T01 -> T02 -> T03 -> T04 -> T05 -> T06 -> T07 -> { T08 ... T16 } -> T17
```

Within the parallel block there is one ordering constraint: T13 edits the import half of `e2e/import-categories.spec.ts` after T12 lands.

## 17. Open decisions

Defaults are in place, so work can begin; the developer can override them.

| # | Decision | Default in this spec | Alternative |
| --- | --- | --- | --- |
| 1 | Primary button color | Black or white `default`, with green `brand` for the single key action | All primary buttons green with a black label |
| 2 | D-011 and D-012 wording | The document takes qualities only; T01 updates `docs/design.md` to say so | The developer rewrites the decisions |
| 3 | UI SDK | Not installed while beta | Accept a beta exception, against the dependency rule |
| 4 | Product mark | A new simple inline SVG mark in brand green | Keep the arrow, which conflicts with its expense meaning |
| 5 | Tutorial progress | `localStorage`, per user and device | A server table to sync across devices (needs a migration and an endpoint) |
| 6 | Tutorial first-run scope | Core path, with the rest optional | Every screen in one long run |
| 7 | Default theme | System | Force dark |

## 18. Not verified

- **No wireframes were made.** The OpenPencil app was not running, so the auth and shell layouts are specified as text diagrams only.
- The `components.json` above was not run. `style`, aliases and generated import paths must be checked with `shadcn@4.23.0` in T02, because the project uses relative imports and the existing primitives were adapted by hand.
- Colors were measured by formula and by the palette validator. They were not tested on physical displays or with people who have color vision deficiency.
- Screen reader behavior of `HelpHint` and the step card is specified from guidance and must be tested with NVDA and VoiceOver.
- The Portuguese and Spanish copy was written with care but has had no native review.
- The behavior of the Wails webview with the fonts and the tutorial was not tested.
- Bundle-size impact of the fonts was not measured.

# Design

vukem-finances has one visual language. This document is the brief and a short token reference; the full specification (tokens, layouts, components, contextual help, keyboard hints, the tutorial and the migration plan) is [design-system.md](design-system.md), and where the two differ that one wins. The code lives in `resources/css/tokens.css` and `resources/js/components/`.

## Feel

Calm, dense and numeric. The screen is quiet so that the numbers stand out: tabular figures in Geist, tight alignment, white, black and one brand green, `rgb(0, 158, 0)`, and no decoration that does not carry meaning. A person should read a month of money at a glance, with a keyboard or a mouse.

## Originality

The design is original (decisions D-011 and D-012). Another product may inspire a quality, such as restraint, density or clear type, and never code, assets, copy, identity or composition. No screenshot, copy, name, asset or layout of another product is stored in this repository, and none is to be added. If a reference seems necessary, ask the author first.

## Rules

1. **Color is never the only signal.** Income and expense always carry a sign or a word as well as `--vf-income` or `--vf-expense`. `MoneyText` writes the sign and an `aria-label` such as "minus 12 dollars 50".
2. **Contrast meets WCAG 2.2 AA**: 4.5:1 for text, 3:1 for control borders, focus ring and icons. `resources/js/design/contrast.test.ts` measures every pair listed below in both themes and fails on a miss.
3. **Focus is always visible.** One ring (`--vf-focus`, 2 px, 2 px offset) for every focusable element, and a `Highlight` outline in forced-colors mode.
4. **Touch targets are at least 44 px** (`--vf-target-min`) on coarse pointers.
5. **Motion is short and optional.** Durations come from tokens. Under `prefers-reduced-motion` the spatial shift goes to zero and opacity and color transitions remain.
6. **Dark mode follows the `dark` class** on `<html>`, set before paint in `resources/views/app.html`; `color-scheme` follows it.
7. **Everything is reachable by keyboard and screen reader**, with shortcuts that never fire while the person types (see `docs/shortcuts.md`).

## Tokens

| Token | Light | Dark | Use |
| --- | --- | --- | --- |
| `--vf-bg` | `#ffffff` | `#000000` | page background |
| `--vf-surface` | `#ffffff` | `#0a0a0a` | cards, popovers, dialogs |
| `--vf-surface-muted` | `#f5f5f5` | `#171717` | muted blocks, hover |
| `--vf-border` | `#e5e5e5` | `#262626` | hairlines and separators (decorative) |
| `--vf-control-border` | `#858585` | `#767676` | borders of inputs and buttons (3:1) |
| `--vf-text` | `#0a0a0a` | `#f5f5f5` | body text |
| `--vf-text-muted` | `#525252` | `#a3a3a3` | secondary text |
| `--vf-text-subtle` | `#6b6b6b` | `#8f8f8f` | tertiary text, still 4.5:1 |
| `--vf-ink`, `--vf-on-ink` | `#0a0a0a`, `#ffffff` | `#f5f5f5`, `#0a0a0a` | the default (black or white) button |
| `--vf-brand` | `#009e00` | `#009e00` | fills, icons, chart marks (3:1; never normal text on light) |
| `--vf-on-brand` | `#000000` | `#000000` | label on the brand fill |
| `--vf-brand-text` | `#007a00` | `#2fc42f` | brand-colored text and links |
| `--vf-income` | `#007a00` | `#2fc42f` | positive amounts (with a sign) |
| `--vf-expense` | `#b42318` | `#ff8f86` | negative amounts (with a sign) |
| `--vf-warning`, `--vf-info` | `#8f5600`, `#1d5fc4` | `#f0b94a`, `#7ab0ff` | status text (with an icon and a word) |
| `--vf-focus` | `#007a00` | `#2fc42f` | focus ring |

Hover, active, subtle and chart tokens are in the specification. Spacing (`--vf-space-1` to `--vf-space-24`), the type scale (`--vf-text-*`, mapped to Tailwind `text-h1`, `text-body`, `text-caption` and the rest in `finance.css`), radius, shadows, motion and layers are in the same file. Geist Variable and Geist Mono Variable are self-hosted from `@fontsource-variable`. The shadcn-style variable names used by `components/ui/` (`--background`, `--primary`, `--ring`, ...) are aliases of these tokens.

## Components

`PageHeader`, `MoneyText`, `AmountInput`, `DataTable`, `Tabs`, `Pagination`, `Toast`, `Combobox` and `Kbd` sit next to the primitives in `resources/js/components/`. The frame is `layouts/Shell.tsx`: a skip link, a left sidebar fed by `resources/js/nav.ts`, a header with the command palette and shortcut help buttons, and `main#content`.

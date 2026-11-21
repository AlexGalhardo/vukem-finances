# Design

vukem-finances has one visual language. This document is the brief and the token reference; the code lives in `resources/css/tokens.css` and `resources/js/components/`.

## Feel

Calm, dense and numeric. The screen is quiet so that the numbers stand out: tabular figures, tight alignment, one brand hue, and no decoration that does not carry meaning. A person should read a month of money at a glance, with a keyboard or a mouse.

## Originality

The design is original (decisions D-011 and D-012). Market products inform which features exist, never how they look. No screenshot, copy, name, asset or layout of another product is stored in this repository, and none is to be added. If a reference seems necessary, ask the author first.

## Rules

1. **Color is never the only signal.** Income and expense always carry a sign or a word as well as `--vf-income` or `--vf-expense`. `MoneyText` writes the sign and an `aria-label` such as "minus 12 dollars 50".
2. **Contrast meets WCAG 2.2 AA**: 4.5:1 for text, 3:1 for control borders, focus ring and icons. `resources/js/design/contrast.test.ts` measures every pair listed below in both themes and fails on a miss.
3. **Focus is always visible.** One ring (`--vf-focus`, 2 px, 2 px offset) for every focusable element, and a `Highlight` outline in forced-colors mode.
4. **Touch targets are at least 44 px** (`--vf-target-min`) on coarse pointers.
5. **Motion is short and optional.** Durations come from tokens and drop to zero under `prefers-reduced-motion`.
6. **Dark mode follows the `dark` class** on `<html>`, set before paint in `resources/views/app.html`; `color-scheme` follows it.
7. **Everything is reachable by keyboard and screen reader**, with shortcuts that never fire while the person types (see `docs/shortcuts.md`).

## Tokens

| Token | Light | Dark | Use |
| --- | --- | --- | --- |
| `--vf-bg` | `#fbfcfb` | `#0e1310` | page background |
| `--vf-surface` | `#ffffff` | `#151b17` | cards, popovers, dialogs |
| `--vf-surface-muted` | `#eff3f0` | `#1d2520` | sidebar, muted blocks, hover |
| `--vf-border` | `#d5ddd8` | `#28332c` | hairlines and separators (decorative) |
| `--vf-control-border` | `#6f8176` | `#6f8176` | borders of inputs and buttons (3:1) |
| `--vf-text` | `#13201a` | `#edf3ef` | body text |
| `--vf-text-muted` | `#475a50` | `#a6b6ac` | secondary text |
| `--vf-brand` | `#1c7448` | `#62d596` | the brand hue: primary actions, links |
| `--vf-on-brand` | `#ffffff` | `#08150d` | text on the brand color |
| `--vf-income` | `#17663c` | `#6fdc9f` | positive amounts (with a sign) |
| `--vf-expense` | `#b3261e` | `#ff8f86` | negative amounts (with a sign) |
| `--vf-focus` | `#1c7448` | `#7fe3ae` | focus ring |

Spacing (`--vf-space-1` to `--vf-space-8`), type scale (`--vf-text-xs` to `--vf-text-xl`), radius (`--vf-radius`) and motion (`--vf-duration-fast`, `--vf-duration-base`, `--vf-duration-slow`) are in the same file. The shadcn-style variable names used by `components/ui/` (`--background`, `--primary`, `--ring`, ...) are aliases of these tokens.

## Components

`PageHeader`, `MoneyText`, `AmountInput`, `DataTable`, `Tabs`, `Pagination`, `Toast`, `Combobox` and `Kbd` sit next to the primitives in `resources/js/components/`. The frame is `layouts/Shell.tsx`: a skip link, a left sidebar fed by `resources/js/nav.ts`, a header with the command palette and shortcut help buttons, and `main#content`.

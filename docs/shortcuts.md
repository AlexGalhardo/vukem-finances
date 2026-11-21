# Keyboard shortcuts

One registry (`resources/js/shortcuts/registry.ts`) owns every shortcut. A feature registers its actions with `useAction`; it never writes a key handler. Every action is also in the command palette (`Ctrl+K`, `Cmd+K` on macOS, or the header button) and in the shortcut help (`?`), so everything stays reachable by mouse and screen reader.

## Rules

- Shortcuts never fire while the person types: in `input` (text-like types), `textarea`, `select`, `contenteditable`, elements with `role` `textbox`, `combobox` or `searchbox`, `data-shortcuts="off"`, and anything inside them. They also ignore IME composition, key repeat, events already handled, and any modal dialog other than the palette and the help.
- Key notation: `g d` is a sequence (one second between keys), `?` a single key, `mod+k` a chord. `mod` is Cmd on macOS and Ctrl elsewhere.
- Browser-reserved combinations are refused by `registerAction` (`resources/js/shortcuts/reserved.ts`).
- WCAG 2.1.4: the preference "Single-key shortcuts" (default on, `localStorage` key `vf.shortcuts.single`) turns off every binding without a modifier. Chords with `mod` stay.
- `/` is registered once, by `QuickAdd` (mounted on every signed-in page): on `/transactions` it focuses the search field, anywhere else it goes to `/transactions#search`, which focuses the field on arrival. Typed inside a field it is just a character.
- A feature epic may add rows to the table below. It may not take a key that is already listed.

## Allocation

| Keys | Action id | Owner |
| --- | --- | --- |
| `mod+k` | `open-palette` | shell |
| `?` | `open-shortcut-help` | shell |
| `[` | `toggle-sidebar` | shell |
| `g d` | `go-dashboard` | shell |
| `g t` | `go-transactions` | shell |
| `g a` | `go-accounts` | shell |
| `g c` | `go-categories` | shell |
| `g b` | `go-budgets` | shell |
| `g i` | `go-import` | shell |
| `g w` | `go-workspace` | shell |
| `n e` | `new-expense` | epic 07 |
| `n i` | `new-income` | epic 07 |
| `n t` | `new-transfer` | epic 07 |
| `/` | `search-transactions` | epic 08 |
| `x` | `export` | epic 12 (reserved) |
| `g r` | `go-reports` | epic 12 (reserved) |
| `g k` | `go-cards` | shell |
| `g u` | `go-recurrences` | shell |
| `g g` | `go-goals` | epic 11 (reserved) |

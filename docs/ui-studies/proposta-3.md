# Proposal 3 · Ledger Line

Independent design study · 2026-11-24 · Web, desktop, mobile and public landing.

Ledger Line puts the current period, the scope of the figures and the next
action on one alignment grid. A persistent context strip above the register is
the signature of the design. Its totals align with the amount column, so the
eye can move between an individual entry and the aggregate without changing
reading direction. Use cobalt for actions and selection, green for positive
values and burgundy for expenses. The approved freedom to replace the primary
green applies to this study.

## Evidence and design decisions

- Issue selection in a keyboard-driven work tool exposes actions through both
  a command menu and direct interaction. In Vukem, put contextual actions on
  the row and keep their equivalents discoverable through the existing
  registry where supported. [Issue interaction reference](https://linear.app/docs/creating-issues#select-issues)
- A register can summarize a transaction at the current account scope and
  disclose its related entries on demand. Use a concise transfer row and show
  source and destination together in detail. [Account register reference](https://code.gnucash.org/docs/C/gnucash-guide/txns-registers1.html)
- Reconciliation documentation shows the value of naming the scope of a
  total. Apply that principle to import previews and filtered views by
  explicitly labelling their counts and totals. [Reconciliation reference](https://actualbudget.org/docs/accounts/reconciliation/)
- Sticky controls can obscure focus and the minimum AA target size is 24 CSS
  px with exceptions. Use fully visible focus and a 44-unit touch target as
  design requirements. [WCAG 2.2](https://www.w3.org/TR/WCAG22/)
- The project already specifies tabular figures, a shortcut registry, a native
  desktop save boundary and offline mobile persistence. The proposal builds
  on those constraints. [Design brief](../design.md) · [Current system](../design-system.md) ·
  [Shortcuts](../shortcuts.md) · [Desktop](../desktop.md) · [Mobile](../mobile.md)

The first sketch treated the dashboard as a set of independent metrics.
The register layout makes the period and scope visible beside each aggregate
and gives dashboard, reports and transactions a shared visual grammar.
The research informs interaction qualities. The layout, palette and example
data below were created for this study.

## Editable tokens

[Token specification](proposta-3.tokens.json) uses flat theme objects for color
integration. Shared numeric dimensions are CSS px on web and desktop and
logical units on mobile. Type dimensions follow the platform font scale.

| Color role | Light | Dark |
| --- | --- | --- |
| `bg` | `#F2F5F8` | `#111925` |
| `surface` | `#FFFFFF` | `#192435` |
| `ink` | `#142338` | `#F0F5FD` |
| `muted` | `#435570` | `#B6C5DB` |
| `border` | `#CED7E3` | `#374861` |
| `accent` | `#214FCC` | `#9BB6FF` |
| `onAccent` | `#FFFFFF` | `#111D38` |
| `soft` | `#E3EBFF` | `#263B66` |
| `positive` | `#166044` | `#8AE0B3` |
| `negative` | `#99263B` | `#FFA4B2` |
| `warning` | `#764800` | `#FFD081` |
| `subtle` | `#52627A` | `#9CACCA` |
| `controlBorder` | `#697D98` | `#8196B3` |
| `hover` | `#EBF0F6` | `#223149` |
| `accentHover` | `#193FA8` | `#B2C7FF` |
| `accentPressed` | `#143485` | `#C4D5FF` |
| `focus` | `#214FCC` | `#9BB6FF` |
| `info` | `#244B91` | `#ACC7FF` |
| `infoSoft` | `#E6EFFF` | `#243858` |
| `positiveSoft` | `#E3F4E9` | `#1A3D33` |
| `negativeSoft` | `#FCE9ED` | `#492A39` |
| `warningSoft` | `#FFF0D1` | `#473719` |
| `transfer` | `#435570` | `#B6C5DB` |
| `chart1` | `#214FCC` | `#9BB6FF` |
| `chart2` | `#166044` | `#8AE0B3` |
| `chart3` | `#99263B` | `#FFA4B2` |
| `chart4` | `#764800` | `#FFD081` |
| `chart5` | `#644399` | `#D9B4FF` |
| `scrim` | `#101B2B` | `#070B12` |

- Use `bg` for the shell canvas, `surface` for reading planes, `hover` for
  pointer feedback, and `soft` with a 3 px leading accent rule for selection.
  Add a checkmark or the appropriate selected/current accessibility state.
- `border` is a decorative separator. `controlBorder` identifies input
  boundaries. Use accent pairs for primary buttons across default, hover and
  pressed. Secondary actions use ink on surface. Use muted text on hover
  fill for disabled controls plus disabled semantics and a visible reason
  when the restriction needs explanation.
- Status foregrounds pair with their matching Soft fills. Use a sign or
  label with every monetary value, including neutral transfers. Format with
  locale and currency. Keep a currency with its amount and align figures
  right across a column. Read values as localized words where supported.
- Use chart colors in the listed order on surface. Pair series with text and
  dashes, symbols or pattern. Use border for optional gridlines and muted for
  axis text. Give every chart a summary and a tabular alternative.
- The minimum computed sRGB contrast for text on bg, surface, hover and soft
  is 5.19:1 light and 4.83:1 dark. The minimum for controlBorder, focus and
  accent against those same surfaces is 3.53:1 and 3.66:1. Primary button
  labels across all three states reach 6.87:1 and 8.38:1. Matching status
  foreground and fill pairs reach 6.59:1 and 6.70:1. Chart colors against
  surface reach 6.87:1 and 7.83:1. These are token checks. Rendered focus,
  forced colors and charts require prototype verification.

| Family | Values and application |
| --- | --- |
| Typography | Geist locally bundled with system fallback. Regular 400, medium 500, semibold 600. Size/line height 12/16 caption, 13/18 label, 14/20 body, 16/24 mobile body, 18/24 section, 26/32 title, 36/40 total. Tabular lining figures. |
| Spacing | 0, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64. Rows use 12 horizontal padding. Prose stays within 68 characters per line. |
| Corners and lines | 6 control, 10 panel, 14 dialog, 999 pill. Divider 1, focus 2 with offset 2, selection rule 3. |
| Layout | 48 top bar, 208 sidebar, 56 collapsed rail, 344 inspector, 1440 max content. Gutters 16/20/24. Form max 560. Rows 36 dense, 44 standard, 56 touch minimum. Pointer target 32 minimum, touch target 44 minimum. |
| Elevation | Only menus, dialogs and overlay sheets cast a shadow, using scrim at 18% with 0/8/24/0 offset and blur. Modal backdrop uses scrim at 55%. Layers 0 base, 10 sticky, 20 menu, 30 modal, 40 toast. |
| Icons | Lucide shapes, 16 inline, 20 controls, 24 touch, 1.75 stroke. Accessible names on icon-only buttons. |
| Motion | 0 ms focus and immediate states, 90 ms feedback, 140 ms overlay with cubic-bezier(0.2, 0, 0, 1). Reduced motion and native mobile use 0 ms. |
| Theme | Initial web paint and desktop webview use the existing pre-paint theme mechanism and system fallback. Respect platform appearance on mobile and system appearance on landing. Use system Highlight focus in forced colors. |

## Shell and fast paths

```text
[Navigation] | Transactions                         [New transaction]
             | Nov 2026 · All accounts     Filtered net   −R$ 184,60
             | [Search] [Account] [Category] [Clear filters]
             | Date   Description           Category           Amount
             | 24 Nov Market                Food              −R$ 82,60
             | 23 Nov Bus fare              Transport         −R$ 12,00
```

The example amounts are fictional and the visible rows represent a slice of
the filtered set. Use an explicit scope label beside any aggregate. Summaries
depend on server-provided totals at that scope. If only paged rows are
available, identify a page total or omit the aggregate.

- At 1200 and up, use the full sidebar and an adjacent inspector when the
  register retains at least 600 px. The inspector opens with the selected
  transaction identity, then amount, date, account, category and actions.
  Preserve filter query, page and scroll when closing it. A row action opens
  it, and Escape closes it and restores focus to the opener.
- From 768 to 1199 use a labelled-on-focus icon rail with a labelled expand
  control and overlay inspector. Below 768, use a navigation drawer and a
  full-height detail view. Below 420 or at high zoom, stack scope and summary.
  Keep horizontally scrollable data regions named and keyboard reachable
  when a table needs full width. Preserve amount and description in the first
  mobile reading pass.
- Keep the existing nav registry as the source for the sidebar and palette.
  Show Dashboard and Transactions first, then Money, Planning, Insights,
  Data and Settings. Keep Account and Documentation in the footer and theme
  controls last in the top bar. Show account/workspace context above the
  grouped links when the surface supports it.
- Keep command palette `mod+k`, search `/`, transactions `g t`, expense
  `n e`, income `n i`, transfer `n t` and help `?`. Use registered
  names and hints, with visible buttons for all actions. Keep existing IME,
  editing and modal guards and the single-key preference. Suggestions retain
  their documented local bindings. Avoid adding an independent key listener.
- Use one scroll container for the main register and make the context strip
  sticky where it fits. Give focus targets a scroll margin covering all
  sticky chrome. Move focus to the page heading on navigation. Announce
  filter counts when settled without stealing focus from search.
- Desktop uses the same shell within the native window, including narrow
  window rules. First launch uses the native-gated Setup flow. Native File
  menu holds backup and data-folder commands. CSV/JSON use the native save
  dialog, with distinct completion and cancellation. External links open in
  the system browser. [Desktop behavior](../desktop.md)
- Native mobile keeps Home, Transactions, Add, Accounts and More. Keep Add
  visibly labelled and open a full-screen amount-first form using the
  locale keyboard. Transactions use date-grouped two-line rows with the
  signed amount at the trailing edge and detail on tap. Filters open via a
  labelled control and summarize active choices in the header. Honor safe
  areas, large text and visible actions for VoiceOver and TalkBack.
  [Native behavior](../mobile.md)
- Personal uses the implicit workspace with setup or invitation as
  appropriate. SaaS shows the workspace switcher, role-aware actions and
  enabled billing and connected-app entries. Keep pending deletion and role
  restrictions visible near the relevant controls. Desktop is personal.
  Mobile has its own local UI and optional sync to a configured server.
  A workspace name on a paired phone identifies the sync target.
  [Project edition rules](../../AGENTS.md)

## Patterns and state contracts

| Pattern | Layout and states |
| --- | --- |
| Register | PageHeader, context strip, search and filters, DataTable, Pagination. Use stable column widths and table headers. Row menu and inspector show equivalent actions. Distinguish first-use empty with a creation CTA, filtered empty with Clear filters, and load error with Retry and retained filters. |
| Quick entry | Begin with expense, income or transfer, then amount, account, date and the applicable details. Show source and destination together for transfers. Keep entered values after validation or server failure. Put focus on the first invalid field, use inline error text, and prevent duplicate submits while saving. Announce success once and restore focus. |
| Changes and deletion | Give irreversible actions an explicit verb, entity and consequence. Use a dialog with Cancel reachable first for high-risk actions. Reuse password and exact-name checks where required. Use inline confirmation for persistent states and a toast for transient success. |
| Import and review | Separate file, preview and result. Preview displays account, accepted count, errors and duplicates where reported. Keep error rows navigable. Suggestion review shows the proposed category and confidence with explicit accept/reject. |
| Settings | Use SettingsSection rows with title, summary and action. Keep forms in the reading column. Give unavailable integrations a plain explanation when the setting is discoverable, gated routes follow server availability. Token creation shows the secret once, with copy feedback and clear expiry/revocation status. |
| Data loading | Preserve the shell and headings, mark updating regions busy and reserve dimensions. Keep prior content only while its scope is clearly labelled. Show no-data and zero as separate cases and use locale-aware empty placeholders. |
| Auth and consent | Use AuthLayout and first-error focus. Explain verification, expired invites and recovery beside the applicable action. Consent names the requesting host, workspace and scopes with separate allow and deny actions. |
| Native sync | Distinguish saved on device, queued offline, syncing, synced, sign-in needed and workspace deleted. Show queue count where available and use a polite live announcement for status changes. Compare competing values with source and time where provided. Merge and Replace have explicit outcomes, with export available before replacement. |

## Page inventory and treatment

The web inventory counts 39 page components from `resources/js/pages/`,
including shared error and generic admin templates. Routes and availability
come from `routes/`, page definitions from `app/pages/`, and sidebar
membership from `resources/js/nav.ts`. Dialogs are part of their parent
screen. This inventory includes pages beyond the tutorial list in the
existing design system.

| Web components | Proposed treatment and key state |
| --- | --- |
| Dashboard | Period and scoped totals above recent activity and upcoming items. First-use view leads to account setup or entry. |
| Transactions | Canonical register. Filtered-empty, long descriptions, transfers, pending refresh, row detail and pagination. |
| Accounts, Cards, CardStatement | Align balances and limits on the trailing edge. Show archived account and card states where supported. Statement header uses its billing period and status. Keep payment and its confirmation in statement context. |
| Categories, Budgets, Goals, Recurrences | Category hierarchy and archive status. Budget rows show spent, limit and remaining with an over-limit label. Goals show target and contribution progress. Recurrences show next date and paused state. |
| Reports | Keep period, basis and comparison above the figures. Align comparison columns, distinguish zero from missing and keep exports in report context. |
| Import, Suggestions | Follow the import/review pattern and expose a clear result when there are no eligible rows. |
| Workspace, Devices, Tokens, ConnectedApps | Members and invitations with role. Devices show pairing expiry and revocation. Tokens and connected apps show scope and revoke actions. ConnectedApps is SaaS and follows remote availability. |
| AI, Notifications, Settings/Telegram | AI opt-in, missing key and disabled features. Notifications show read state and preferences. Telegram shows link status and code expiry where configured. |
| Billing, Admin/Revenue | SaaS billing distinguishes plan, entitlement and pending/expired/paid PIX status. Admin revenue is separately gated and uses a scoped ledger treatment. |
| Setup, InviteAccept, InviteRegister | Personal first-owner setup, invitation identity, expiry and completion. On desktop the gate changes the setup code requirement. |
| Auth/Login, Auth/Register, Auth/CheckEmail, Auth/VerifyEmail, Auth/ForgotPassword, Auth/ResetPassword | Signed-out templates with consistent labels and validation, SaaS public registration and verification, neutral recovery confirmations and expired-link recovery. |
| Auth/TwoFactor, Auth/TwoFactorChallenge, Auth/AccountDeletion, Privacy, OAuthConsent | Enrollment and challenge, backup codes where applicable, deletion grace date and re-auth, consent and denial. Keep destructive and recovery actions visible and distinct. |
| Error, Admin/Dashboard, Admin/ResourceList, Admin/ResourceShow | Error status and recovery action. Admin uses AdminLayout, clear resource context and permission-aware actions. |

The native inventory has 17 route screens and two layout files in
`mobile/app/`. Desktop wraps the personal web pages and contributes native
startup, native menu, save, backup and external-link states.

| Native route screens | Proposed treatment and key state |
| --- | --- |
| Tabs index, transactions, add, accounts, more | Period overview, compact register, amount-first entry, account balance list, grouped destinations. Preserve local operation in airplane mode. |
| budgets, cards, categories, recurrences, reports, tags | Stacked rows with aligned values, counts or dates. Reports keep a readable text/table view at large font sizes. |
| settings | First-run choices, lock availability, backup share and confirmations for destructive actions. |
| sync, sync-connect, sync-merge, sync-conflicts, sync-problems | Status, pairing, merge/replace, value comparison and retry with explicit recovery for expired or revoked access. |

## Public landing

The three locale routes `/`, `/pt-br/` and `/es/` share a composition
based on the register's reading order. Keep paragraphs narrow and align
figures independently of language. Documentation pages are outside this
study. [Site constraints](../site.md) · [Claims register](../landing-brief.md)

```text
Vukem Finances                                   Language   Repository
[One headline about running your own finance app] | [Static register]
[Short statement about where data lives]          | [Caption]
[Installation guide] [Desktop releases]
[Plain statement about hosted-service availability]
```

- At wide sizes use a 5:7 hero split, with copy and links left and a
  locally rendered product specimen right. Use three original fictional
  transactions in a static HTML table to demonstrate date, description and
  signed amount. Label it as illustrative. On small screens place it after
  both links and keep the currency next to the amount.
- Lead with installation documentation and desktop releases. Keep the
  no-hosted-service notice in the first section. Follow with How it works,
  grouped finance tasks, privacy and ownership, editions, installation,
  FAQ via native details, and the approved footer destinations. Use a
  side-by-side edition comparison at wide sizes and stacked headings at
  narrow widths. Plan limits come from `site/generated/plans.json`.
- Give each factual sentence an evidence row in the claims register with
  the same catalog keys across EN, PT-BR and ES. Preserve the current
  exclusions for mobile/sync, MCP, stores, signed builds and account
  deletion. Keep the gate for terms and use the allowed plain-link CTAs.
- Deliver the theme with CSS system preference and local assets. Keep the
  page static with one h1, landmarks, a skip link, visible focus and locale
  metadata. Respect 60 KiB HTML, 40 KiB gzipped CSS and 150 KiB gzipped
  initial total budgets. Use the existing claims and built-output checks
  during implementation. [Static checks](../site.md#commands)

## Prototype acceptance and implementation friction

1. From Dashboard reach Transactions with `g t` or one visible navigation
   action. Search with `/`, open a result and close it with filters and
   scroll intact. Confirm regular keyboard and pointer paths at 1440, 1024,
   768 and 390 CSS px. At 200% zoom the focused target remains visible.
2. Open expense entry in one shortcut sequence or one direct action. Save
   a valid example and return focus. Simulate a rejected save and verify
   field values and the error remain visible. Compare time and action count
   for these tasks against the existing UI using the same seed.
3. Verify both themes with text scaling, forced colors, a reader and touch.
   Try an extended currency amount and long localized labels. On mobile
   save offline, then review a pending sync conflict. On desktop cancel and
   complete an export through native save.
4. Check both web editions and a viewer role with all relevant feature
   flags. Review the three landing locales at the same breakpoints with
   script disabled, then check claims, weight limits and destinations.

The transaction list already updates the URL through Inertia with
`preserveState` and `replace`. Inspector continuity must be checked across
filter visits and browser history. The evidence is an integration constraint
in the app and does not establish a Vukem framework defect.
[Transactions source](../../resources/js/pages/Transactions.tsx)

The desktop documentation records a Wails integration limitation for
external content and a save bridge for downloads in the embedded webview.
The practical design implication is that the native dialog owns the file
destination and cancellation. The study found no new verified framework
defect through static inspection. [Desktop downloads](../desktop.md#downloads-and-exports)

Research, source inventory and token contrast were checked. Visual and
task-timing criteria above are for implementation and prototype review.

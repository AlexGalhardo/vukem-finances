# Proposal 2 · Monthly Margin

Independent design study for Vukem Finances. November 24, 2026.

## Direction

The signature is a budget line with a visible end mark. A category's spending advances toward the mark, the effective monthly limit sits immediately above it, and the remaining amount or overage sits to the right. The same line appears in the learner's worked example and in the public landing illustration. It makes “how much room is left?” answerable at a glance.

The palette is white and cool slate with indigo controls. Mint, raspberry and amber carry financial or operational meaning. Indigo can mark a selected month or a planned limit. Use neutral ink for balances and values whose sign alone does not tell the story. Sums use tabular figures and align by currency. Reserve raised surfaces for transient UI, and keep the page's rows and section edges aligned to a single column rhythm.

## Research and evidence

- The repository brief calls for calm numeric density, an original composition and keyboard access. Its design specification describes contextual help and an existing tutorial with per-screen anchors. I kept the app's component vocabulary and proposed a budgeting path through those anchors. The brief names no external product, so the following references are my selections. [Design brief](../design.md) · [Design system](../design-system.md) · [Keyboard shortcuts](../shortcuts.md)
- Linear describes reducing the visual prominence of navigation and placing actions consistently. I applied that to quiet permanent chrome and predictable page actions. [Linear design refresh](https://linear.app/now/behind-the-latest-design-refresh)
- Actual Budget explains how its tracking view presents planned and actual values together. I used that pairing for the category line, using Vukem's effective limit and its own carryover semantics. [Actual Budget tracking budget](https://actualbudget.org/docs/getting-started/tracking-budget/) · [Vukem budget calculations](../../app/finance/budgets.go)
- YNAB introduces budgeting through a sequence of concrete choices. I used one category and one sample expense as the onboarding exercise. The proposed lesson teaches Vukem's category limits and preserves its existing transactions and budget operations. [YNAB start guide](https://www.ynab.com/guide/the-ultimate-get-started-guide)
- WCAG 2.2 informed the contrast, focus, resizing and announcement criteria. The geometry, copy, palette and sample below are original to this proposal. [W3C WCAG 2.2](https://www.w3.org/TR/WCAG22/)

## Editable tokens

The [token file](proposta-2.tokens.json) provides flat color roles in each theme and shared scales. Numeric lengths are CSS px at the design baseline and dp on native, with font sizes in scalable units. Type tuples are size, line height, weight and tracking. Web and the desktop WebView use bundled Geist Variable. Native uses the platform face with tabular numerals and the nearest supported weights.

| Token | Light | Dark | Role |
| --- | --- | --- | --- |
| `bg` | `#F3F5FA` | `#111522` | Page canvas |
| `surface` | `#FFFFFF` | `#1A2030` | Ledger and form surface |
| `ink` | `#171D30` | `#F4F6FF` | Headings and amounts |
| `muted` | `#505A72` | `#B8C2DA` | Secondary text and disabled labels |
| `border` | `#D5DAE6` | `#343E56` | Decorative separators |
| `accent` | `#3149B4` | `#AEBEFF` | Primary action, links and selection mark |
| `onAccent` | `#FFFFFF` | `#151E43` | Primary action label |
| `soft` | `#E7EBFA` | `#292F50` | Selected row and guide surface |
| `positive` | `#006447` | `#83DDB8` | Income and positive status |
| `negative` | `#9C2F48` | `#FF9EB5` | Expense, overage and error |
| `warning` | `#774600` | `#FFD08A` | Near limit and caution |
| `info` | `#24578A` | `#9ACDFF` | Neutral status and sync information |
| `control` | `#69738B` | `#8692AE` | Required control outline and chart zero line |
| `focus` | `#3149B4` | `#AEBEFF` | Keyboard focus |
| `hover` | `#EDF0F7` | `#242C41` | Hover and disabled control background |
| `accentHover` | `#293D99` | `#C1CCFF` | Primary hover fill |
| `accentPressed` | `#233482` | `#99ADF9` | Primary pressed fill |
| `positiveSoft` | `#E0F2E9` | `#193A32` | Positive status background |
| `negativeSoft` | `#F9E7ED` | `#492637` | Negative status background |
| `warningSoft` | `#FAEFD9` | `#422F1B` | Warning background |
| `infoSoft` | `#E3EFFB` | `#243A50` | Information background |
| `chartPlan` | `#3149B4` | `#AEBEFF` | Plan mark |
| `chartActual` | `#505A72` | `#B8C2DA` | Actual mark |
| `overlay` | `#171D3099` | `#050812B3` | Modal scrim (RRGGBBAA) |

| Scale | Values and use |
| --- | --- |
| Type | Display 40/44 for a single prominent figure, title 28/34, section 20/26, body 15/22, label 14/20, caption 12/18. |
| Space | 0, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64. Pair labels and fields with 8, stack fields with 16, divide sections with 32. |
| Radius and strokes | 8 controls, 12 panels, 999 pills. Divider 1, focus 2 with 2 offset, budget track 4. |
| Sizes | Rail 224, header 56, guide 320, form 480 max, content 1280 max. Table row min 44. Touch targets min 44 web and 48 native. Icons 20, subordinate 16. |
| Elevation and layers | Single shadow on menus and dialogs. Base 0, sticky 10, menu 20, dialog 30, toast 40. |
| Motion | 100 ms response, 160 ms disclosure, cubic-bezier(0.2, 0, 0, 1). Reduced motion 0 ms. Native review can use immediate transitions. |
| Layout | Under 768, one content column and a menu button. From 768, persistent rail where space permits. At 1200 and above, show the guide beside content after accounting for the rail. |

Contrast is the minimum across each listed set of solid backgrounds, calculated with WCAG relative luminance. Use a surface-colored gap around focus on filled controls. Keep dividers decorative and use `control` for required outlines and the unfilled budget track. Charts use direct labels, distinct marks and a data table. Apply colors only to the listed surface pairings.

| Foreground | Backgrounds | Required | Light min | Dark min |
| --- | --- | --- | --- | --- |
| `ink` | `surface`, `bg`, `hover`, `soft` | 4.5 | 14.07 | 12.04 |
| `muted` | `surface`, `bg`, `hover`, `soft` | 4.5 | 5.80 | 7.28 |
| `accent` | `surface`, `bg`, `soft` | 4.5 | 6.44 | 7.18 |
| `onAccent` | `accent`, `accentHover`, `accentPressed` | 4.5 | 7.66 | 7.47 |
| `positive` | `surface`, `bg`, `positiveSoft` | 4.5 | 6.19 | 7.68 |
| `negative` | `surface`, `bg`, `negativeSoft` | 4.5 | 6.07 | 6.69 |
| `warning` | `surface`, `bg`, `warningSoft` | 4.5 | 6.92 | 8.85 |
| `info` | `surface`, `bg`, `infoSoft` | 4.5 | 6.42 | 6.99 |
| `control` | `surface`, `bg`, `hover` | 3 | 4.16 | 4.46 |
| `focus` | `surface`, `bg`, `soft`, `hover` | 3 | 6.44 | 7.18 |
| `chartPlan` | `surface`, `bg` | 3 | 7.02 | 8.98 |
| `chartActual` | `surface`, `bg` | 3 | 6.32 | 9.10 |

## Learning and guided planning

**Entry.** Offer “Plan your first month” on the dashboard and from the tutorial center. The existing tutorial supplies progress, skip, reopen and fallback when an anchor is unavailable. Reframe the welcome around an outcome with a single primary action and a visible “Explore on my own” exit. In a populated workspace the same entry can say “Review this month”. Remember progress with the existing per-user tutorial persistence.

**Understand.** Use the selected month, then show this illustrative category in the guide before opening the real budget. Use the active locale for amounts, dates and speech. The worked example uses pt-BR and entirely fictional data.

```text
Orçamentos                                    Novembro de 2026

Mercado                                             Restam R$ 180
R$ 420 gastos                      Limite do mês R$ 600
━━━━━━━━━━━━━━━━━────────│
```

The end mark denotes 100% of available budget. Put any carried amount in the arithmetic detail, for example “R$ 550 de limite + R$ 50 do mês anterior”. Use the server or native budget summary and avoid adding a second finance rule. At zero spending expose “R$ 0 gastos”. At the effective limit show “Limite atingido”. Above it cap the solid bar at the end mark and place a text label such as “R$ 60 acima” beside it. At zero effective limit use a text state and omit the ratio where unavailable. An amber “Perto do limite” can appear at 80% as a display threshold with no alert side effect.

**Set a budget.** Move through the existing month, progress and form anchors in that order. If categories are empty, link to Categories and offer to return. Explain rollover beside the existing switch. Save requires the user's explicit form action. A viewer can follow the worked example and read the current budget. Suppress steps for writes when the role or deletion state blocks them.

**Record and revisit.** Offer the existing quick-entry or transaction action, with the category explicit in the instruction. Save remains user-initiated. Lead back to Budgets for the same month and category, then show where to find the monthly report. If the selected period is empty, distinguish no limits from no spending and from a filter that hides results. Where mobile cannot support the full tour, use short inline help at the same decision points. For planning, teach the limits and spending views first.

**Layout and focus.** The web guide lives beside the ledger on wide windows and in a bottom sheet or inline content on narrow viewports. Preserve anchored controls, keep one interactive step in view and return focus to the invoking control on dismissal. The native guide appears inline as part of First Run and the Budget empty state, with system back and visible dismissal. Treat the web guide panel and persistent resume behavior as prototype proposals to test against the existing engine.

## Component recipes and states

| Area | Recipe and variants |
| --- | --- |
| Shell | Skip link, one page H1 targeted after navigation, existing nav groups and visible current-page label with marker and `aria-current`. Keep theme buttons last in the top bar and Account and Documentation in the footer. Keep palette and shortcuts from the existing registry, hints optional on touch and suppressed while typing. |
| List and ledger | Use DataTable on signed-in web pages. Align amounts right, date and description left, show category alongside its icon. Narrow lists pair the signed amount with the description and put date and category on a second line. Preserve sorting semantics, pagination and filter reset. Keep full descriptions available at larger text scales. |
| Money and charts | MoneyText and the native accessible amount use signs or words. Transfers use paired direction and account names with neutral ink. Budgets and goals use labelled end marks. Reports label plan with a dashed line or hollow mark and actual with a solid line or filled mark. Use direct labels and a tabular view for multi-category plots instead of a swatch-only legend. Retain category names and existing supported category icons. |
| Fields | Label above, short help below, error adjacent to input and associated programmatically. Use existing AmountInput and locale-aware formatting. Keep entered values on error and focus the first invalid field. Pending save uses a stable-width label, blocks duplicates and announces completion. |
| Controls | Primary action uses accent/onAccent with explicit hover and pressed variants. Secondary uses surface/control/ink. Disabled uses hover/muted with native disabled semantics. Destructive confirmation uses negativeSoft/negative with an explicit verb and scope. Focus visible on every keyboard control and system Highlight in forced colors. |
| Dialog and help | Trap focus in modal, Escape closes where safe, return focus to opener. Tooltip names an icon, inline hint explains decisions and a sheet holds optional material. Keep essential information readable without hover. |
| Feedback | Loading holds the eventual content footprint without shimmer. Partial load keeps current data and marks the pending portion. Success toast confirms the action once, permission restrictions appear near the affected area, and recoverable errors offer a focused retry and preserve input. Empty datasets suggest the first allowed action. Filtered-empty states offer clearing the filters. |
| Restricted states | Pending-deletion and viewer surfaces keep readable data and label their write restrictions. Name user-visible consequences before privacy deletion, device revocation, disconnect or erasure. Use only actions available to the current role, edition and deployment. |

## Page inventory and screen direction

Inventory from the route definitions and React page components. The named surfaces below also identify the frames for reviewing patterns in both themes. Root is a redirect. Downloads, webhooks, polling, API and callback routes support flows. [Routes](../../routes) · [Page declarations](../../app/pages) · [React pages](../../resources/js/pages) · [Navigation](../../resources/js/nav.ts)

| Web page or route | Proposed focus and states |
| --- | --- |
| Dashboard · `/dashboard` | Period and balance first, monthly flow and upcoming items below, quick-entry action in its existing context. Guide entry near the period summary, with an empty first-run variant and a populated review variant. |
| Transactions · `/transactions` | Filters above the ledger, counts and pagination below. Entry, edit, transfer, split or installment flows retain their available controls. Show filtered empty, validation and pending save. |
| Accounts · `/accounts` | Net worth above account list, with account type and balance. Archived grouping, first account and confirm archive flows. |
| Cards and CardStatement · `/cards`, `/cards/{card}` | Limit beside utilization, statement period beside the total. Open, closed and paid where supported, with payment confirmation and amount. |
| Categories · `/categories` | Income and expense grouping, icon plus name, tags in their existing area. Empty, edit and any applicable plan limit. |
| Budgets · `/budgets` | The signature month view. Show month, category, amount spent, effective limit, remainder or overage and rollover detail. Include zero spend, exact limit, overspent, no limit and viewer states. |
| Goals · `/goals` | Progress towards target with amounts and contribution action. Empty, active and reached. |
| Recurrences · `/recurrences` | Next occurrence first, then frequency and amount. Show active and paused where supported, with empty and edit. |
| Reports · `/reports` | Stable period and section tabs, series or breakdown paired with accessible table, comparison period distinct from current. Include no data, export and narrow viewport. |
| Import · `/import` | File, mapping when applicable, preview and confirmation. Show invalid rows and duplicates in the preview and outcome where available. Keep valid input during correction. |
| Suggestions and AI · `/suggestions`, `/ai` | Suggestions show original and proposed category with explicit acceptance. AI settings show enabled state, key presence and limits. Existing draft and assistant overlays show review, pending and recoverable failure. |
| Workspace · `/workspace` | Roles, invitations and export in independent settings rows. SaaS switching and creation follow the current edition and permissions. Pending deletion is an inline read-only state. |
| Notifications and Telegram · `/notifications`, `/settings/telegram` | Unread and read list, preferences and optional connection. Telegram renders only when configured, with waiting, linked and expired code states. |
| Tokens and Devices · `/tokens`, `/devices` | Scope, label and status first. Token or pairing code gets a distinct copyable view and expiry when applicable, plus device revoke confirmation. |
| Privacy and AccountDeletion · `/account/privacy`, `/account/deletion` | Organize personal export, reauthentication, pending deletion, cancellation and local erasure according to edition. Separate destructive semantics from the default action. |
| ConnectedApps and OAuthConsent · `/connected-apps`, `/authorize` | SaaS remote integration when enabled. Consent lists verified origin, workspace and scopes with a visible decline. Connected apps show revocation and empty states. |
| Billing · `/billing` | SaaS only. Plan, owner actions and provider-dependent flows. PIX waiting, paid, expired and retry. Keep text and amount legible independently of the payment graphic. |
| Auth and Setup · `/login`, `/register`, `/register/check-email`, `/email/verify`, `/forgot-password`, `/reset-password/{token}`, `/setup` | One 480-wide form column. Pending, resend, invalid or expired and success states as applicable. Public registration only in SaaS. Personal server setup includes the setup code. Desktop setup uses the verified launch flow and omits that field. |
| Invites · `/invites/{token}`, `/invites/{token}/register` | Workspace identity followed by accept or registration when allowed. Expired, already used and unauthorized states. |
| TwoFactor and TwoFactorChallenge · `/user/two-factor`, `/two-factor-challenge` | Explain setup, challenge and recovery codes at the step where needed. Keep copy and recovery actions explicit. |
| Admin dashboard, resource list, resource detail and revenue · `/admin`, `/admin/{slug}`, `/admin/{slug}/{id}`, `/admin/revenue` | Reuse the ledger, detail and settings typography in the guarded panel. Keep the admin context visible. Revenue is SaaS only. |
| Error | Show problem and next allowed action. Prototype a missing route and an expired session as separate states. |

### Desktop

Use the web personal composition inside native window chrome. At a narrow window width use the same responsive shell breakpoints and keep window controls native. Review first launch, later login, keyboard palette, canceled and successful native saves, opening an external link, and a backup failure. Keep local backup and app-data folder actions in the platform menu and distinguish them from CSV and JSON workspace exports. Preview in an offline state using bundled fonts. Use the platform dialog appearance. [Desktop architecture and UX](../desktop.md) · [Desktop source](../../desktop)

### Mobile

The native app is local-first. Pairing optionally connects a server workspace and shows its current role and sync state. Reuse semantic colors and the budget line with platform text and 48 dp hit targets. Support 200% text by wrapping and stacking money beneath labels when needed. Retain the five existing tab destinations. [Mobile documentation](../mobile.md) · [Native router](../../mobile/app) · [Native screens](../../mobile/src/screens)

| Surface | Direction and states |
| --- | --- |
| FirstRun and LockGate | Language, currency, time zone and first account in a clear sequential setup, followed by contextual budgeting help. Locked screen renders without financial data. Show refusal of lock setup when device authentication is unavailable. |
| Home, Transactions, Add | Home shows balance and monthly flow. Transactions shows a compact ledger and filters. Add exposes expense, income and transfer fields with visible actions, offline saving and validation. |
| Accounts and More | Balances and editing, plus grouped links in More. Use full-word labels and secondary status where needed. |
| Budgets, Categories and Tags | Use the same limit end mark, with the remaining or overage amount wrapping below. Category type, tag names and explicit edit buttons. |
| Cards and Recurrences | Card limits and statement details where supported. Show next date and frequency for recurring entries with labelled edit actions. |
| Reports and Settings | Period summary and readable figures. Language, currency, lock and local backup use rows with predictable actions and native share sheets. |
| SyncStatus and SyncConnect | Status makes offline usable and distinguishes queued changes from failure. Connect shows address and one-use code, connection progress and actionable errors. |
| SyncMerge, SyncConflicts and SyncProblems | Merge and Replace explain what happens to local records, with export before Replace. Conflicts show current and replaced values. Recovery handles rejected changes, re-pair after revocation, and explicit erasure after workspace deletion. |

## Public landing

Apply Monthly Margin to the landing at `/`, `/pt-br/` and `/es/`. The first viewport is a split composition. On the left use one H1 about managing personal finances on your own machine or server, two plain links for installation guidance and source, and the registered no-hosted-service statement. On the right use a static, original worked budget line matching the proposal's training example. It is an illustration with sample data and an adjacent text equivalent. At small widths it follows the CTA in document order. Use inline HTML and CSS with system typography and shared color roles. Follow the system theme with CSS `color-scheme`.

| Section | Visual direction |
| --- | --- |
| How it works | Three sequential rows for setup, recording or importing, and planning. Use numbered steps here because the order matters. |
| Features | Group by everyday tasks with short descriptions from the claim register. Use captioned, local first-party screenshots if retained and distinguish them from the illustrated example. |
| Privacy | A full-width section with the data location and ownership statements grouped beside the relevant evidence-backed caveats. |
| Editions | Parallel columns for Personal and self-hosted SaaS, stacking on small screens. Use generated Free and Pro limits and plain text for the self-hosting explanation. |
| Install | Two paths for Docker instructions and desktop releases with the registered unsigned-build note. |
| FAQ and footer | Native details and summary, visible keyboard focus, repo and documentation links, privacy, terms only when approved, and language switch. |

The marketing page remains static Astro with no form, cookie, API, external resource, hosted-service implication or price. Use only the permitted CTA destinations. Keep English, pt-BR and Spanish catalog keys in parity. Before implementation, map each factual sentence to the claims register. Claims about mobile, MCP, stores, signed builds and account deletion are outside that register. Public documentation and legal copy are outside this design study. [Static site constraints](../site.md) · [Landing structure and claims](../landing-brief.md)

## Prototype sequence and checks

| Review | Frames and pass criteria |
| --- | --- |
| First planning session | Wide web 1440, narrow web 390 and native 390, in both themes. Without instruction from a moderator, locate the month, interpret the example, find the budget form and explain rollover. Try the same with a viewer. |
| Follow-up and recovery | Overspent row, empty month, 200% text, form validation, signed amount announcements, keyboard-only progress and modal dismissal. The guide preserves focus and advances only on explicit user action. |
| Platform and edition | Personal server setup, desktop 1024-wide first run and native save, SaaS onboarding and workspace context, mobile offline with queued changes. Check available actions against existing gates. |
| Landing | 1440 and 390 in both themes, with pt-BR and Spanish wrapping. Hero links navigate directly, illustration reads without color and the page works without scripts. Verify claims and built output when implemented. |

This study verifies color calculations and inventories files. Visual rendering, assistive technology and task completion require prototype review.

## Vukem friction

The Go page declarations, React files, navigation and tutorial are separate views of the surface inventory. The declarations are typed and the tutorial owns its own anchors and visibility rules. A framework-level inventory that exposes declared Inertia components alongside route bindings could reduce manual reconciliation. Framework tooling remains untested in this study. Mobile's theme export from CSS is already a project-level bridge. Its generated 44 dp target differs from the 48 dp accessibility contract in the mobile docs, so the proposal explicitly targets 48 dp on native. [Page declaration contract](../../app/pages/pages.go) · [Tutorial inventory](../../resources/js/tutorial/screens.ts) · [Generated mobile tokens](../../mobile/src/theme/tokens.ts) · [Mobile accessibility](../mobile.md#accessibility)

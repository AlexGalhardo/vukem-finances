# Proposal 1 — Ledger Trace

Independent design study for Vukem Finances, 24 November 2026. The reference checkout is `9013ebf`. This is a design handoff with synthetic examples. The repository's product, site and shared design specifications are outside the change set.

## Direction

Build each finance screen around a question someone can answer in under a minute. Which transactions explain this number? Which dates and accounts does it cover? Where can I check the underlying data? Use an **evidence line** as the recurring visual device: a horizontal rule with a short blue inset that joins the amount, the scope it represents and the next available way to inspect it. On a wide ledger the line continues across to the selected entry; on a phone it precedes the evidence in reading order. A rule carries meaning only when accompanied by a label or linked control. Ordinary separators stay neutral.

Use cool paper, white entries and deep ink in light mode. Use a blue-black canvas, a raised slate surface and pale blue actions in dark mode. Blue expresses navigation, selection and inspection in both themes. A negative amount is signed and may use burgundy; warning and destructive controls have their own icon and explicit verb. Calm comes from a single alignment axis for amounts and predictable reading order, with color concentrated in the current scope. Keep the product wordmark. The supplied scope allows a new primary color.

I reduced the initial dashboard idea from a grid of equal metrics to a period block and one continuous ledger. A grid made a balance, a movement and a projected value look interchangeable. The proposed hierarchy gives those values stable names and lets the reader follow the data used to calculate each one.

## Research and provenance

| Observation from the source | Design inference for Vukem |
| --- | --- |
| GnuCash asks for statement date and ending balance and exposes a separate difference before completion. [GnuCash manual](https://www.gnucash.org/docs/v5/C/gnucash-manual/gui-reconcile.html) | Show period and basis next to an amount. Compare like with like. The prototype covers reviewing existing records. Persistent reconciliation, statement matching and a posted adjustment would require product work. |
| Actual distinguishes a ledger balance from the amounts checked against the bank and makes that distinction visible during review. [Actual Budget reconciliation](https://actualbudget.org/docs/accounts/reconciliation/) | Use independent labels for local save, server sync and import outcome. Apply a verified or matched label only when the underlying source supplies that state. |
| Carbon documents table sorting, expansion and actions as independent behaviors. [IBM Carbon data table](https://www.carbondesignsystem.com/building-blocks/core/components/data-table/guidelines) | Give sorting a header control and inspection a link with a name. Preserve the row's position and the current filters when evidence opens or closes. |
| WCAG 2.2 describes minimum contrast, non-text contrast and focus visibility including obstruction. [W3C WCAG 2.2](https://www.w3.org/TR/WCAG22/) | A sticky summary gets scroll clearance. Selection and focus use distinct treatments. Model 44 px coarse targets in web and 48 dp targets in native. |

The references supplied interaction principles. The composition, palette, wireframes and examples below were developed for this proposal. Repository rules come from the [design brief](../design.md), [design system](../design-system.md), [shortcut register](../shortcuts.md), [mobile guide](../mobile.md), [desktop guide](../desktop.md), [site guide](../site.md) and [landing brief](../landing-brief.md). Where the design system describes a migration baseline, the current code supplies the inventory.

## Tokens and component rules

[Editable JSON](proposta-1.tokens.json) defines all visual variables for the proposed frames. The `light` and `dark` groups contain individually editable six-digit hex values. Geometry and typography are shared. Use CSS px or native dp for geometry and respect scaled text. Core color roles:

| Role | Light | Dark | Application |
| --- | --- | --- | --- |
| bg | `#F2F6F9` | `#101923` | Canvas and shell |
| surface | `#FFFFFF` | `#172432` | Ledger, detail, dialog |
| ink | `#172638` | `#F1F6FB` | Primary text and most amounts |
| muted | `#4C5E70` | `#B2C1D2` | Date, metadata and secondary copy |
| border | `#CBD6E0` | `#354759` | Decorative rules |
| accent / onAccent | `#194E79` / `#FFFFFF` | `#A7CEFA` / `#12243A` | Primary action and label |
| soft | `#E4EDF6` | `#233447` | Hover and scope background |
| selected | `#D7E7F6` | `#293F57` | Selected row with rail and state |
| positive | `#146247` | `#8BDEB6` | Positive result or completed operation |
| negative | `#962D40` | `#FFADBA` | Negative result or dangerous operation |
| warning | `#784700` | `#F7C279` | Review needed |
| focus / controlBorder | `#245E95` / `#6A7D90` | `#B6D7FF` / `#8299B0` | Focus outline and input boundary |

The registered text pairs measure at least 5.29:1 light and 5.89:1 dark. The registered non-text pairs measure at least 3.36:1 light and 3.67:1 dark. These are conservative lower bounds from the WCAG relative-luminance calculation. Use `controlBorder` anywhere a boundary carries meaning. Give the 2 px focus outline a 2 px gap to expose the contextual surface. In forced colors use the system highlight and retain a textual selected state. Limit chart fills and category swatches to their registered surfaces. Differentiate chart series through labels, solid/dashed strokes, and markers. Series colors are measured against their background. Use whitespace or a contrasting outline where marks would otherwise touch.

| Pattern | Composition and behavior |
| --- | --- |
| Type and money | Geist 16/24 body, 14/20 table, 12/16 metadata, 20/28 section, 28/36 title and 36/44 total. Use Geist Mono for codes or references. Native can fall back to the system typeface. Use lining tabular figures and right-align whole amounts. Keep currency and sign with the value. Avoid ellipsis in amounts. Let rows grow with text. |
| Shell | Retain `PageHeader`, grouped sidebar and theme buttons last in the top bar. Sidebar footer contains Account and Documentation. Place the active workspace next to scope controls in SaaS. A workspace switch clears the prior workspace's data from view while loading. Settings use `SettingsSection`. |
| Ledger | `DataTable` with 48 px minimum rows or 40 px in fine-pointer dense views. Keep description and amount readable first. Period, account and category controls appear above the list. Display result count with scope. Keep visible pagination separate from totals. Use column headers and text for sorted direction. |
| Evidence panel | Title from the selected entry, exact amount and date, account and category, then supported details and actions. With sufficient width place it beside the ledger. At narrower widths let it open as a sheet or full-screen route. Preserve filters, page and scroll on close and return focus to the inspected control. |
| Buttons and forms | One primary button per local task. Secondary controls use an outline; contextual actions use text and icon. Press uses `accentPressed`, hover uses `accentHover` for filled buttons. Reserve danger fill for final destructive confirmation. Put a label above each input, with hint or validation below. Use structured fields for amount, date and account. Preserve drafts after a failed save where the existing form allows it. |
| Feedback | Use a named status with icon and text. Inline feedback handles actionable errors. Use a brief toast for completed non-critical actions. On missing or stale evidence, keep the reason beside the figure. Reserve a skeleton's geometry at initial load and mark the region busy. During refetch, keep prior rows only while their scope still matches and label them as updating. |
| Help and keys | Use `HelpHint`, `TermHint`, `useAction` and registry-generated hints. Keep `mod+k`, `g t`, `/`, `x` and `n e` as already allocated. Respect typing, modal and IME guards and the single-key preference. Keep help reachable through explicit controls and the tutorial's existing hooks. |
| Movement | No count-up animations. Use 100 ms feedback and 160 ms disclosure on web/desktop. Remove spatial motion when requested. Native movement tokens are zero. Use elevation for overlays. Lucide on web/desktop and native controls with the same meanings on mobile. |

Use signed or named amounts with `MoneyText` on web and spoken money labels on native. Format exact cents using the product currency and locale. Represent zero as zero, a missing value as “Unavailable” and a pending value as “Calculating…”. Display relevant date basis in reports, including comparison periods. Keep aggregates supplied by the product. An account balance, a sum of the filtered result, and a subtotal for the visible page each need their own label and data. In a multi-currency model, present currency groups unless an existing conversion provides the basis. A transfer keeps its source and destination and uses the product's transfer accounting. A purchase and a card payment retain their different kinds.

## Layout and edition matrix

| Surface | Layout | Product gates and exception states |
| --- | --- | --- |
| Web 1200 px and wider | 224 px sidebar, main ledger, 288 px optional evidence rail within a 1440 px content maximum. Gutter 32 px. | Full work area according to active edition and role. Maintain scroll margin below sticky chrome. |
| Web 768–1199 px | Gutter 24 px, collapsible navigation and sheet for evidence. At narrower widths within this band, prioritize the ledger before opening the sheet. | Follow existing permission checks on write controls. |
| Web below 768 px | Gutter 16 px, compact header, responsive list with date and amount visible. Filters move into a named sheet with applied count and clear action. | Keep existing web routes and capabilities. Expose all primary actions to touch. |
| Desktop | Reuse personal web views within the Wails native window. Adapt to the window width and preserve keyboard focus. | Setup uses the native gate path. Native menus own backup and opening the data folder. Export invokes native save. Treat user cancellation as a normal result. An external resource opens in the system browser. |
| Mobile local | Home, Transactions, Add, Accounts, More. Use a two-line row group with the signed amount aligned right, 64 dp minimum row and 48 dp targets. At 200% text size allow currency to occupy its own line. | First run, device lock, local backup and offline transaction entry. Present saved-to-device feedback where relevant. |
| Mobile paired | Same five tabs with connected workspace named in Sync screens. Local save and sync are separate states with a queue count when available. | One paired workspace, sync while app is open. Respect server capability, plan and role. Read-only pairing leaves inspection available and explains why changes are unavailable. |
| Personal web and desktop | One implicit workspace. Setup, sign-in and invitation paths use the product's personal branches. | Desktop uses only the personal edition. The personal shell omits workspace creation, switching, billing and admin revenue. |
| SaaS web | Workspace context is visible in the ledger, export and destructive prompts. | Public signup verifies email. Membership, plan, owner access and optional configuration govern actions. Remote consent and connected apps follow server registration and HTTPS. Pending deletion displays the existing deadline and read-only state. |

Theme selection follows System, Light or Dark where the application offers that preference. Apply the chosen theme before initial paint on web, use the app's native theme setting on mobile, and match the Wails surface to the web theme while the OS owns native dialogs. Use the same focus and semantic colors in both editions.

## Page inventory and design treatments

Web inventory combines the [typed page registry](../../app/pages/pages.go) and [React page tree](../../resources/js/pages). It includes 38 typed pages plus Error. Names below map to source files. Native inventory comes from the [route tree](../../mobile/app) and the [screen tree](../../mobile/src/screens). Layout files are navigation scaffolding.

| Web pages | Screen treatment and key states |
| --- | --- |
| Dashboard | Period evidence line, opening/closing values only where supplied, movement summary and recent entries. Empty starts with first account or entry. |
| Transactions | Filters, results, amount column, inspected entry, paging, export. Distinct first use, no match, failed query and permission state. |
| Accounts, Cards, CardStatement | Account identity and balance, card status, billing cycle and statement amount, itemized movements and payment state. |
| Categories, Budgets, Goals, Recurrences | Kind and category hierarchy, planned and actual, target and contribution, next occurrence and enabled status. Empty or completed states use explicit text. |
| Reports | Period and comparison controls, main figure, chart and named data table, visible units and export choices where available. |
| Import, Suggestions | File, mapping and review stages with separate outcome counts. Suggestions show original and proposed category, available confidence and explicit accept/reject actions. |
| Notifications, Workspace | Unread and read history, preferences with named channels, and role/member rows. A pending workspace keeps a persistent read-only explanation. |
| AI, Settings/Telegram | Owner/config gating where applicable, enabled status, consent or connection state and appropriate error/revoke feedback. |
| Tokens, Devices, ConnectedApps, OAuthConsent, Auth/TwoFactor | Scope and expiry, one-time copy, pairing, revocation and connected app identity. Explicit allow/deny scope and workspace at consent. Show recovery where the flow supports it. |
| Billing | Plan and owner actions. Separate pending PIX, confirmed, expired and failed results using server state. Clear contact or retry guidance from the available action. |
| Privacy, Auth/AccountDeletion | Explain export and deletion scope before confirmation. Show grace-period deadline and cancellation when available. |
| Setup, InviteAccept, InviteRegister | Explain where the data or membership belongs. Cover invalid setup code where required, expired invite and role context. |
| Auth/Login, Auth/Register, Auth/CheckEmail, Auth/VerifyEmail, Auth/ForgotPassword, Auth/ResetPassword, Auth/TwoFactorChallenge | Single narrow column. Use explicit status for errors, verification and recovery, focus the first invalid field and label every control. |
| Admin/Dashboard, Admin/ResourceList, Admin/ResourceShow, Admin/Revenue | Preserve separate admin navigation and access. Use scoped list/detail treatments, privileged confirmations, explicit billing status and period. |
| Error | Render the available status, readable reason and a valid route back. |

| Native pages or gates | Screen treatment and key states |
| --- | --- |
| Home, Transactions, Add, Accounts, More | Period summary and latest records, sticky date group, keyboard-friendly entry, account balance with movement context, and grouped secondary destinations. |
| Budgets, Cards, Categories, Recurrences, Reports, Tags | Compact readable data rows, named progress, status and next dates. Keep charts secondary to readable figures. The existing Cards flow hosts statement information. |
| Settings, FirstRun, LockGate | Preferences and backup, initial locale/currency/time zone/account, and an opaque locked state before any data mounts. |
| SyncStatus, SyncConnect, SyncMerge | Show offline queue, syncing, synced and connection or pairing errors. Merge or replace explains impact and offers export before replacement. |
| SyncConflicts, SyncProblems | Show server value and local replaced value with field names. Separate conflict, queued error, credentials required, deleted workspace and read-only access. |

Desktop inherits available personal React pages. Its native inventory includes the Wails window, first launch, another-instance focus, system save dialog, File menu backup and data folder, plus external-browser handoff. See the [desktop guide](../desktop.md) and [desktop shell](../../desktop/shell.js).

## Reference frames for FigCraft

Use a fictional November period. All examples are BRL with `pt-BR` formatting. The scoped movement example is +R$ 3.200,00 − R$ 1.245,70 = +R$ 1.954,30. Use the same amounts across themes and editions so visual comparisons expose differences in hierarchy.

```text
Large web / desktop: 1440 x 960

[Navigation]   Movimentações                        [Nova despesa]
               Novembro de 2026     Conta principal
               ━━━ Entradas +R$ 3.200,00   Saídas −R$ 1.245,70   Movimento +R$ 1.954,30
               [Buscar] [Período] [Conta] [Categoria]       [Exportar]
               Data    Descrição / Categoria        Valor │ Mercado Central
               24 nov  Mercado Central           −R$ 82,40 │ −R$ 82,40
               23 nov  Aluguel                −R$ 1.163,30 │ 24 nov 2026
               20 nov  Salário                +R$ 3.200,00 │ Conta principal
               [3 movimentações]                           │ Alimentação
```

Prepare both personal and SaaS versions. The SaaS frame adds the workspace picker ahead of the scope controls. A viewer variant retains filters and allowed exports and removes the write action. The period summary uses a product aggregate supplied for that same scope, or is omitted in the transaction list and illustrated on Reports. The evidence panel shows supported details using the selected row. Its persistence is a presentation proposal. Transfers and exact transaction totals should follow the current report definitions.

```text
Mobile: 390 x 844

Movimentações                        [Filtros]
Novembro de 2026 · Conta principal
━━━ 3 movimentações

24 nov
Mercado Central                    −R$ 82,40
Alimentação · Conta principal
23 nov
Aluguel                         −R$ 1.163,30
Moradia · Conta principal

[Início] [Movimentações] [Adicionar] [Contas] [Mais]
```

Use the same row order at 390 and 320 px. At 200% text size place the amount on a following line when required. The Add frame keeps amount and kind first, then account, date and category. Show a separately labeled destination on transfer. With a pending sync queue, the completion state reads “Salvo neste dispositivo” with a path to status. With read-only pairing, use the data view and its permission explanation.

### Audit scenarios

| Scenario | Frames and transitions |
| --- | --- |
| Inspect a month | Dashboard → scoped Transactions or Reports → entry evidence → return to the prior view. Carry only filters supported by the destination, and show the resulting scope there. Include no-match and unavailable totals variants. |
| Review a report | Cash flow shows signed income, expense and net movement with exact values. Comparison includes current and previous dates. Keyboard order reaches the data table and export. Include partial period and zero activity variants. |
| Review a CSV or OFX | File stage → format/mapping → preview → result. Put source line or available source identifier beside each issue. Distinguish accepted, skipped and invalid or duplicate rows only where supplied by the importer. Include unsupported file and partial result. |
| Inspect a statement | Card → statement cycle → items and payment state. Name cut-off and due date separately. Show previous and next cycles only where available. |
| Handle sync | Queued offline → sync in progress → up to date, or actionable error/conflict. Compare exact differing fields and leave an explicit next action. Retain local evidence after revocation until the existing disconnect/erase path applies. |
| Export or delete | Describe scope and format before export. On desktop include native save, cancellation and failure. For an irreversible operation name the target, show the current deadline if any, and anchor initial focus on a safe action. |

## Public landing — From overview to evidence

Scope covers `/`, `/pt-br/` and `/es/`. Compose at desktop 1440 px and phone 390 px in both themes, using system preference and CSS. Public documentation enters only as a link destination.

1. Header with wordmark, locale links and a primary installation link. Keep its height modest so the first evidence image has room.
2. Hero with existing localized value proposition at left and a wide Vukem demo-seed screenshot showing the ledger or report at right. The evidence line joins screenshot caption and task. Use existing hero and screenshot sources. On mobile place text and CTA first, then a legible crop or downscaled screenshot with a caption. An image example remains visibly a screenshot.
3. Follow with product work in reading order. Record entries, inspect reports, then install and choose an edition. Use the existing registered content for features, screenshots, editions, how it works, install, privacy and FAQ. Frame the edition comparison as a single aligned table on wide screens and stacked labeled rows on mobile. Personal and self-hosted SaaS get distinct fit statements and the same visual weight.
4. Keep the existing disclosure of no hosted offering close to the edition comparison and installation. Show unsigned desktop-build language beside the desktop Releases link. Link to the Docker documentation, GitHub repository and GitHub Releases with plain anchors. Footer retains existing privacy and terms gating.

Use a 5/7 copy-to-image split above 1200 px with a 32 px gap. Align the hero CTA baseline with the caption rule beneath the image. The caption identifies the seeded screenshot and the work it illustrates. Below 768 px, use one column with a 16 px gutter. Stack installation choices under their own headings and keep the installation and release links close to their respective descriptions.

| Landing element | Acceptance |
| --- | --- |
| Claims | Every factual sentence maps to a row in the claims register. The original three locales keep parallel copy keys. Free/Pro values come from generated plans. Mobile, MCP, store availability and unregistered capabilities stay out of the copy and screenshots. [Claims register](../landing-brief.md#claims-register) |
| Media | Same-origin assets from the app demo seed, descriptive alt text, explicit intrinsic size, screenshot cap of 150 KB per image. Use inline CSS visual rules and local assets. No third-party requests, forms, authentication, checkout, cookies, analytics or API calls. [Static site rules](../site.md) |
| Navigation and states | One H1, logical landmarks, skip link, hreflang and focus-visible anchors. Use native disclosure for FAQ if needed. In dark mode keep image captions readable and image legibility independent of surrounding background. Retain a readable description if an image fails. |

## Vukem integration notes and handoff checks

The typed Inertia registry is an integration boundary for data behind presentation changes. A new aggregate, import provenance field or direct source link would need Go page props and regenerated TypeScript. The current review can use the existing data. [Typed pages overview](../../app/pages/pages.go)

The desktop download path is an observed source-level friction in the Vukem/Inertia web surface hosted in Wails. The app-owned shell intercepts link clicks for native save and reports failure with a hardcoded English browser alert. That path sits outside localized app feedback. Attribute this to the desktop adapter and WebView boundary. The source review found no confirmed Vukem framework defect. [Desktop shell](../../desktop/shell.js)

The mobile palette and geometry use an app-owned CSS-to-TypeScript export. The checked-in export's `targetMin` is 44 dp, while the mobile primitives independently define `TOUCH_MIN = 48` in line with the mobile guide. FigCraft uses 48 dp for native and 44 px for coarse web targets. [Mobile tokens](../../mobile/src/theme/tokens.ts) · [Mobile primitives](../../mobile/src/components/ui.tsx) · [Mobile accessibility](../mobile.md#accessibility)

Review FigCraft frames in both themes at 1440, 1024, 390 and 320 px, with a dense ledger, a single empty ledger and the viewer variant. Check reading and focus order, return from evidence, 200% text size, signed amounts in all three locales, visible sticky focus and reduced motion. Confirm that the desktop native window and save dialog, and native screen-reader behavior, receive a later runtime pass. This handoff contains a source inventory and contrast calculations; the frames are specifications.

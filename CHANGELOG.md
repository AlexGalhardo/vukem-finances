# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Release sections are generated from the commit history with
[git-cliff](https://git-cliff.org) and the configuration in
[.config/cliff.toml](.config/cliff.toml). Print the next one and paste it
below, in place of `[Unreleased]`:

```sh
git cliff --config .config/cliff.toml --offline --unreleased --tag vX.Y.Z --strip header
```

## [Unreleased]

## [0.6.0] - 2026-11-22

### Added

- **site:** Add the self-contained static Astro site project ([`290f362`](https://github.com/AlexGalhardo/vukem-finances/commit/290f362579c60d4c30f924e27db36a9f1fe1c9c6))
- **site:** Configure the fully static build for finances.vukem.dev ([`5f47296`](https://github.com/AlexGalhardo/vukem-finances/commit/5f472961f404764e1e7959f69f4900ca15ca8f45))
- **landing:** Add the claims register, catalogs and forbidden-phrase check ([`42da804`](https://github.com/AlexGalhardo/vukem-finances/commit/42da804b21859736461e1eb8cec2f08ebde1a56e))
- **landing:** Add the landing page in English, Portuguese and Spanish ([`f21cc24`](https://github.com/AlexGalhardo/vukem-finances/commit/f21cc249d61c0fba6b69d6abc605b0f105e423fa))
- **landing:** Add the privacy notice and the gated terms draft ([`62f7130`](https://github.com/AlexGalhardo/vukem-finances/commit/62f7130404965833832511a0ea53d218ebf34585))
- **site:** Add the build checks, the sitemap script and the site tests ([`7f05ae1`](https://github.com/AlexGalhardo/vukem-finances/commit/7f05ae1c2f476124c7653d840544ff3f47a41c88))
- **landing:** Add the product pictures from the demo seed and the share image ([`a062d0d`](https://github.com/AlexGalhardo/vukem-finances/commit/a062d0d82218f7408272d3e3891d32c7fe26b542))
- **site:** Add the docs layout, Markdown renderer and sidebar ([`eab9326`](https://github.com/AlexGalhardo/vukem-finances/commit/eab9326e69e8bc70df39ecf5f74004c39b0057ea))
- **site:** Publish a Markdown twin of every page and llms.txt, and check the structure ([`d229a49`](https://github.com/AlexGalhardo/vukem-finances/commit/d229a4948949c6d0e8dbceefca0b72613925586a))
- **site:** Generate the REST, CLI, MCP, environment, shortcuts and changelog references ([`2e94fba`](https://github.com/AlexGalhardo/vukem-finances/commit/2e94fba3a66a5478bfe7ed57bcb329557275bfd0))
- **site:** Show the last update of every docs page from git history ([`2127cca`](https://github.com/AlexGalhardo/vukem-finances/commit/2127ccafe96fa6955ae32be6f9c4f6406a9b39e2))
- **site:** Fail the build when a docs page names something that does not exist ([`e9a53c0`](https://github.com/AlexGalhardo/vukem-finances/commit/e9a53c0a568c6ef2c6aa80c2f4c68bab8d1a2e9b))
- **privacy:** Add the deletion state, purge date and audit tables ([`d8fb33f`](https://github.com/AlexGalhardo/vukem-finances/commit/d8fb33fe77b23197e1858af2bf03438158cd3481))
- **privacy:** Export the personal data without secrets ([`24c9990`](https://github.com/AlexGalhardo/vukem-finances/commit/24c9990854e78b4d61cceb6bb3dc87474a0609d7))
- **privacy:** Request and cancel the deletion of a workspace ([`ce3e80a`](https://github.com/AlexGalhardo/vukem-finances/commit/ce3e80a8db988fb139ecabdd88c058d15a92aa3d))
- **privacy:** Refuse writes while a workspace waits to be purged ([`e14fbe8`](https://github.com/AlexGalhardo/vukem-finances/commit/e14fbe813930d27576a86a6a6e9420a3d1258421))
- **privacy:** Request and cancel the deletion of an account ([`8919962`](https://github.com/AlexGalhardo/vukem-finances/commit/8919962f329cefc068700fca07e1acc7a6eaba8d))
- **billing:** Cancel the subscription of a workspace that is being deleted ([`aadf4ad`](https://github.com/AlexGalhardo/vukem-finances/commit/aadf4ad10215b99a76567a9b91b30f1c6ae7d635))
- **privacy:** Purge workspaces and accounts after the grace period ([`9dd8df1`](https://github.com/AlexGalhardo/vukem-finances/commit/9dd8df10aaa3bd213a8ecd9491b8ab2d8d6a5af3))
- **sync:** Tell a phone its workspace was deleted and erase it on confirm ([`f02ac4a`](https://github.com/AlexGalhardo/vukem-finances/commit/f02ac4a357d2ba6b6826dc821b4c2e29bb358231))
- **privacy:** Remove the local database files for the personal edition ([`f96009c`](https://github.com/AlexGalhardo/vukem-finances/commit/f96009c7afa11cd6d582e52a22876b5b0fd5fd56))
- **privacy:** Add the privacy page, the account deletion page and the emails ([`f95e270`](https://github.com/AlexGalhardo/vukem-finances/commit/f95e2702854d3ca4ad1416f9a1c883dcfaac1dba))
- **privacy:** Erase all local data from the personal edition ([`9047c39`](https://github.com/AlexGalhardo/vukem-finances/commit/9047c39cffafedfbda7a0382c487d87d23871b53))
- **ui:** Move the theme buttons to the top bar and slim the sidebar footer ([`0d5502d`](https://github.com/AlexGalhardo/vukem-finances/commit/0d5502deb6444ced3dfa6d4b99093910e30ecd5f))
- **ui:** Add the design tokens, Geist fonts and live system theme ([`1270d44`](https://github.com/AlexGalhardo/vukem-finances/commit/1270d4473a9e0bb91dd1ad5bafdd231dfff60817))
- **ui:** Restyle the primitives and add select, popover, checkbox and more ([`7a26c1c`](https://github.com/AlexGalhardo/vukem-finances/commit/7a26c1cbd9e76f23a66968b8b3850830d12d1cc8))
- **ui:** Restyle the shared domain components and add Stat ([`aad3a91`](https://github.com/AlexGalhardo/vukem-finances/commit/aad3a91eb1942a0d65d659db5bb54712d3568a51))
- **ui:** Show registry shortcuts on buttons and add the help and tutorial actions ([`b4d4fec`](https://github.com/AlexGalhardo/vukem-finances/commit/b4d4fec4af0487e88a0da1f07713002755da7565))
- **ui:** Add HelpHint, TermHint, HelpSheet and the help glossary ([`4f2ca19`](https://github.com/AlexGalhardo/vukem-finances/commit/4f2ca19e3ef13f7e85b0b54cbc67af201b1958e1))
- **ui:** Rebuild the app shell and add the auth layout ([`6f4dfcc`](https://github.com/AlexGalhardo/vukem-finances/commit/6f4dfcc9f5afac22ab8796a6c41407eb7c868867))
- **ui:** Add the tutorial engine, copy in three languages and the welcome card ([`72162f4`](https://github.com/AlexGalhardo/vukem-finances/commit/72162f4d6ee671ff2f2f4d4538cfb025759f1bfd))
- **ui:** Rebuild the Dashboard and tag the Reports tour targets ([`e417c11`](https://github.com/AlexGalhardo/vukem-finances/commit/e417c114eb9effb330a38444b37915b73f09352e))
- **ui:** Restyle Transactions with MoneyText, export menu and tour targets ([`e1b446c`](https://github.com/AlexGalhardo/vukem-finances/commit/e1b446cf8d0441294a94cd51f1105c3ad2ec7504))
- **ui:** Restyle Accounts, Cards and CardStatement with Progress and status icons ([`1be6d0b`](https://github.com/AlexGalhardo/vukem-finances/commit/1be6d0b1dc2a698bf91e8ab4da7c7c5a1979a43d))
- **ui:** Restyle Categories, Budgets and Goals with Progress and tour targets ([`fcd4e30`](https://github.com/AlexGalhardo/vukem-finances/commit/fcd4e303b273e37e9225c60f241f1400c96b079b))
- **ui:** Restyle Recurrences, Import and Suggestions and cover the screens in e2e ([`c5b8677`](https://github.com/AlexGalhardo/vukem-finances/commit/c5b867729fb607d1a6884da7e42a736a5d14beb1))
- **ui:** Filters sheet and list layout on Transactions, remaining hints and tour e2e ([`a8baeb3`](https://github.com/AlexGalhardo/vukem-finances/commit/a8baeb321731bad55b2595f542effc138d3653b3))
- **ui:** Rebuild the auth screens on the new layout with focus on the first error ([`a26fb5e`](https://github.com/AlexGalhardo/vukem-finances/commit/a26fb5e36aa7e9e7f523cda21f7e47947a4c12da))
- **ui:** Rebuild workspace, billing, notifications and telegram as settings sections ([`0e47a38`](https://github.com/AlexGalhardo/vukem-finances/commit/0e47a38f654ed971ad9f1f3e84917817d4ac34f8))
- **ui:** Rebuild AI, tokens, devices and two-factor as settings sections ([`20501ed`](https://github.com/AlexGalhardo/vukem-finances/commit/20501ed9f33affe2c337b8c643f41d181afa5b08))
- **admin:** Rebuild the landing page and the admin panel screens on the design tokens ([`5b3d353`](https://github.com/AlexGalhardo/vukem-finances/commit/5b3d3534f0d822b4d7224baa1f94d7f37a7254b6))
- **ui:** Show the running release in the bottom right corner ([`c488be6`](https://github.com/AlexGalhardo/vukem-finances/commit/c488be6129317ecc92815cfc8ff0ebd3129e68bc))

### Changed

- **db:** Add covering indexes for the balances, totals and net worth reads ([`77d125f`](https://github.com/AlexGalhardo/vukem-finances/commit/77d125ff8f7258c640f4b905d622a7ab35c306e0))

### Fixed

- **security:** Refuse JavaScript front matter in the docs pages ([`6cffdb4`](https://github.com/AlexGalhardo/vukem-finances/commit/6cffdb4f661112573ac09d3a8b3ca92d9f795b97))
- **ai:** Refuse alternative spellings of a local host in the base URL ([`216a45b`](https://github.com/AlexGalhardo/vukem-finances/commit/216a45b550cd5f95e89f148814f91142c79412cf))
- **import:** Read a Windows-1252 bank CSV as text instead of keeping invalid UTF-8 ([`ed88915`](https://github.com/AlexGalhardo/vukem-finances/commit/ed889150a2543fd456bbbd2908ff35fc8c7adfe6))
- **recurrence:** Honor an end date that is the zero time ([`91b1b53`](https://github.com/AlexGalhardo/vukem-finances/commit/91b1b53d31cbba2dbee0200a8ece29414c6b8f36))
- **security:** Keep sessions in the database so logout ends a copied cookie ([`9a1229d`](https://github.com/AlexGalhardo/vukem-finances/commit/9a1229d7c50a8e96e618fd13b17ec41756b622b4))
- **privacy:** Refuse a cancel after the grace period and end the process on a failed erase ([`7f314ef`](https://github.com/AlexGalhardo/vukem-finances/commit/7f314ef025a2425826394de261cf4cef6c02a0d7))
- **privacy:** Recheck the same deletion request before purging a workspace ([`5de56ed`](https://github.com/AlexGalhardo/vukem-finances/commit/5de56ed1113ffd4202edc2b60516287094c96359))
- **ui:** List rows no longer overlap on phones and polish found in the screen review ([`432a343`](https://github.com/AlexGalhardo/vukem-finances/commit/432a34381d26f0861e295d0ab2fdd0b4092c9696))
- **finance:** Cut a long description key at a clean boundary ([`e4a2b97`](https://github.com/AlexGalhardo/vukem-finances/commit/e4a2b976721f0f30481d9d65bdd4a95841ec8bec))
- **mobile:** Follow the new palette and use the readable green for text ([`26bfecf`](https://github.com/AlexGalhardo/vukem-finances/commit/26bfecf32235e87b5fcb4df4e93eec979338bf5f))
- **db:** Make a full migrate:rollback run from a fresh database ([`96d08fa`](https://github.com/AlexGalhardo/vukem-finances/commit/96d08fa2eb0337bbc59b155f262f3482452719b8))
- **billing:** Keep the AbacatePay webhook secret out of logs with SecretPaths ([`a78d721`](https://github.com/AlexGalhardo/vukem-finances/commit/a78d721351d0cadb553501f1d5c4306b64b49ca4))
- **sync:** Count the month quota by a date range so the index answers it ([`75b998f`](https://github.com/AlexGalhardo/vukem-finances/commit/75b998f4c88e66ec07bda09f7791f8269d1f001a))
- **import:** Run queued imports again at boot ([`15cbdbc`](https://github.com/AlexGalhardo/vukem-finances/commit/15cbdbc50de4efec498fe409499147f0d962bac7))
- **ci:** Read the budget column of the budget table only ([`9b40dc5`](https://github.com/AlexGalhardo/vukem-finances/commit/9b40dc5380aca5023906c735dcfeffa7c17965e9))

### Security

- **site:** Write the Integrate, Operate and Contribute pages ([`827ad65`](https://github.com/AlexGalhardo/vukem-finances/commit/827ad65c08ba57665dd2f99c2a1b30cb7ac39fe7))

## [0.5.0] - 2026-11-22

### Added

- **cli:** Generate commands and flags from the operation registry ([`2f91064`](https://github.com/AlexGalhardo/vukem-finances/commit/2f91064ab9e8a5e5117942337f5acf4c3ad7b100))
- **cli:** Add local mode through Invoke, refused outside the personal edition ([`f74ac82`](https://github.com/AlexGalhardo/vukem-finances/commit/f74ac822f364097508c383ba569d634512815bc5))
- **cli:** Add remote mode with login, logout and token storage ([`efede62`](https://github.com/AlexGalhardo/vukem-finances/commit/efede62e0420f685a293b8212858d5c7238fc271))
- **cli:** Add table, JSON and CSV output with exact amounts ([`6752c18`](https://github.com/AlexGalhardo/vukem-finances/commit/6752c184efd533b3bbfcc73445c57c52800d10a5))
- **cli:** Confirm destructive commands and require --yes without a terminal ([`a3141ec`](https://github.com/AlexGalhardo/vukem-finances/commit/a3141ecae6cab573872d61e16b0193ca0852322b))
- **cli:** Add bash, zsh, fish and PowerShell completion ([`79415e1`](https://github.com/AlexGalhardo/vukem-finances/commit/79415e1caba655bb64ad02dda0fea86b45040f37))
- **cli:** Generate the CLI reference from the registry ([`a6656c5`](https://github.com/AlexGalhardo/vukem-finances/commit/a6656c54a2b8ca6f8d49d18bdeea113ba685a4ad))
- **desktop:** Ship a macOS disk image and fail the release upload when an installer is missing ([`1088f1f`](https://github.com/AlexGalhardo/vukem-finances/commit/1088f1ff84c2c2e62f19d437d1cb63468597a6ba))
- **desktop:** Package the Windows app as an MSIX ([`de906d6`](https://github.com/AlexGalhardo/vukem-finances/commit/de906d696cf3d36aa1080425d4eaf6712dff4657))
- **ci:** Test the MSIX on every run and submit it to the Microsoft Store on a release ([`a46edca`](https://github.com/AlexGalhardo/vukem-finances/commit/a46edcac179fcd0f4a9e39b3552e5492ebfb47fe))
- **mobile:** Scaffold the Expo app with pinned SDK 58 packages ([`504a729`](https://github.com/AlexGalhardo/vukem-finances/commit/504a729156000e42ceec84727a670205ba218f85))
- **mobile:** Port parseAmount, formatMoney, newUid and addMonths ([`be416e2`](https://github.com/AlexGalhardo/vukem-finances/commit/be416e2e884597808bbb75ddbce2215d3247c1d5))
- **mobile:** Add the local SQLite schema, migrations and outbox repositories ([`23b9c02`](https://github.com/AlexGalhardo/vukem-finances/commit/23b9c02bd49304b721e87c5ecba227db5a4d62a9))
- **mobile:** Add balances, transfers and category rules over the repositories ([`5b05251`](https://github.com/AlexGalhardo/vukem-finances/commit/5b052519220083059eeca083e107e6b838c24ce2))
- **mobile:** Add budgets, cash flow and category reports ([`41f0d62`](https://github.com/AlexGalhardo/vukem-finances/commit/41f0d62d540132fe016a15f8d2c57ab49bd3ce8e))
- **mobile:** Add the recurrence engine, generation and installments ([`46b02fb`](https://github.com/AlexGalhardo/vukem-finances/commit/46b02fb0b0841219819c373a298b9c4e804ed7b9))
- **mobile:** Add card statements and export their vectors from Go ([`3426649`](https://github.com/AlexGalhardo/vukem-finances/commit/34266495fa2131c412fc4f28839181c745b0e8ff))
- **mobile:** Generate theme tokens from the CSS and add the en, pt-BR and es catalogs ([`8306a92`](https://github.com/AlexGalhardo/vukem-finances/commit/8306a921be6596f0e12178545dca06ef8b0a899c))
- **mobile:** Add the shared components with role and label tests ([`2a1596a`](https://github.com/AlexGalhardo/vukem-finances/commit/2a1596a038f3ae4a53a772d851a792a1c49e4725))
- **mobile:** Add the first run, home, add and transactions screens ([`dc68687`](https://github.com/AlexGalhardo/vukem-finances/commit/dc6868754368ee53a1f9e2934c9b6cf5acbc2d7e))
- **mobile:** Add the accounts, cards, categories, tags, budgets, recurrences and reports screens ([`f40104b`](https://github.com/AlexGalhardo/vukem-finances/commit/f40104b51c1a10f6983f71ce5a310bdb6a1f95dd))
- **mobile:** Add settings, the app lock, the local JSON backup and the no-network guard ([`75ebb1a`](https://github.com/AlexGalhardo/vukem-finances/commit/75ebb1a604a9a7488a2674e28c9bd99fde2326fe))
- **sync:** Pair a phone with a short single-use code ([`bf70a3c`](https://github.com/AlexGalhardo/vukem-finances/commit/bf70a3c91b488a743c2469a6db4294acbc58ff0f))
- **sync:** Add the HTTP transport, the address rules and secure token storage ([`331133c`](https://github.com/AlexGalhardo/vukem-finances/commit/331133ceced0f0d62dd6c6861841fabbf69cc0dd))
- **sync:** Add the outbox batches, the sync state and the sync tables ([`340b2ce`](https://github.com/AlexGalhardo/vukem-finances/commit/340b2cefb37ac07fe13b72ab07c7eff04049a3f7))
- **sync:** Apply pulled pages with pending-edit protection and an atomic cursor ([`ad38660`](https://github.com/AlexGalhardo/vukem-finances/commit/ad386603aa95e72958f89f88ecd0e8ba879e7f6a))
- **sync:** Handle push results, re-keying duplicates and rejections ([`6d11f5f`](https://github.com/AlexGalhardo/vukem-finances/commit/6d11f5f057a23f136e80c3c8b4619df4dd6e2f82))
- **sync:** Add the integrity pass and the resync that keeps dirty rows ([`0a877ef`](https://github.com/AlexGalhardo/vukem-finances/commit/0a877efc46937ce12a82e66a8b8306edb047b00c))
- **sync:** Add the sync engine with states, triggers, a mutex and backoff ([`3d28317`](https://github.com/AlexGalhardo/vukem-finances/commit/3d283173bb5fe3d68234aaf3055023098be7a59a))
- **sync:** Add connect and the first-connection merge choice with name adoption ([`2714101`](https://github.com/AlexGalhardo/vukem-finances/commit/27141012f621685831b10e35764b728b441cb9be))
- **sync:** Add the sync strings in en, pt-BR and es ([`d35244b`](https://github.com/AlexGalhardo/vukem-finances/commit/d35244bc9ef35d813b9e2757bfccea516e330706))
- **sync:** Add the Connect, merge choice, status, conflicts and problems screens ([`0714727`](https://github.com/AlexGalhardo/vukem-finances/commit/0714727ae83d6325e210f9a01bd498face3aa21f))
- **operations:** Offer the registry as MCP tools ([`65e649c`](https://github.com/AlexGalhardo/vukem-finances/commit/65e649cb8d221639d2053b550a0b33d39251c2f2))
- **mcp:** Read MCP_TOOLSETS and MCP_ALLOW_DESTRUCTIVE ([`adbc8d7`](https://github.com/AlexGalhardo/vukem-finances/commit/adbc8d78af7228e661e2322dab41412fd3cf2c45))
- **mcp:** Serve the finances over MCP on stdio ([`737768e`](https://github.com/AlexGalhardo/vukem-finances/commit/737768ec8d8409059d143450d57887727afe7570))
- **mcp:** Report progress and stop a cancelled call through its context ([`3839342`](https://github.com/AlexGalhardo/vukem-finances/commit/3839342138fdb03830756c56b15cb7ace69b3113))
- **mcp:** Add MCP to the surface parity test and generate its reference ([`7e36e93`](https://github.com/AlexGalhardo/vukem-finances/commit/7e36e93ac228653aa6c1a36ba694e824cb431e23))
- **oauth:** Add the grant binding table and the framework oauth tables ([`817f51b`](https://github.com/AlexGalhardo/vukem-finances/commit/817f51bfb83f921110e9d3b04b62307f58430a79))
- **mcp:** Let an adapter limit and audit each tool call ([`4c53b9c`](https://github.com/AlexGalhardo/vukem-finances/commit/4c53b9c5944e90c1139a55a0a67d0a0fb7997832))
- **oauth:** Wire the Vukem authorization server to workspaces ([`71723bf`](https://github.com/AlexGalhardo/vukem-finances/commit/71723bf9e68f29fe168404082d7bb0aeb07866b6))
- **mcp:** Serve the tools over Streamable HTTP behind OAuth ([`3871192`](https://github.com/AlexGalhardo/vukem-finances/commit/3871192bb0de0cb2dba5218bff9f774cb191f9e6))
- **oauth:** Serve the metadata, authorization and token endpoints and /mcp ([`b98cc6a`](https://github.com/AlexGalhardo/vukem-finances/commit/b98cc6a5502a8e935b5def58bb8693692ae6b0f1))
- **edition:** Make the personal edition offline-first ([`348ee2e`](https://github.com/AlexGalhardo/vukem-finances/commit/348ee2e6e986b497bcac4cfd4fe0a8f61de6ef7c))

### Fixed

- **workspace:** Install the mail suppression check once per mailer, so concurrent jobs no longer race on it
- **cli:** Refuse a download to a terminal and write --out atomically ([`0b5bddf`](https://github.com/AlexGalhardo/vukem-finances/commit/0b5bddf5e77ec2130e857f86a0416a2b921ccacf))
- **cli:** Harden CSV and table output against terminal and spreadsheet injection ([`1580083`](https://github.com/AlexGalhardo/vukem-finances/commit/15800839984fa7ad9f8f9018105ca9da703a5d9e))
- **cli:** Guard every ;- and tab-separated segment of a CSV cell ([`283f7ee`](https://github.com/AlexGalhardo/vukem-finances/commit/283f7eeea19dafa324e3ea672b42074bf5b2a38b))
- **cli:** Quote every CSV text cell so no reader delimiter can split a formula out ([`65a8b67`](https://github.com/AlexGalhardo/vukem-finances/commit/65a8b6703cb9f35fee92343c2a6c08084d5289cc))
- **cli:** Guard every CSV segment and stream the output ([`d0f5bc2`](https://github.com/AlexGalhardo/vukem-finances/commit/d0f5bc2aec9e8644d4a85c6c347eb3fbe1fa57f8))
- **cli:** Never send an environment token to the stored server ([`641b20e`](https://github.com/AlexGalhardo/vukem-finances/commit/641b20e7a2098da45466c661eb97109cbea08dba))
- **cli:** Guard comma segments of multiline CSV cells ([`f0cf45a`](https://github.com/AlexGalhardo/vukem-finances/commit/f0cf45acfceab7d6e2876abbf9e21cee219fcb0d))
- **cli:** Treat every separator and line break as a CSV guard boundary ([`d0c128a`](https://github.com/AlexGalhardo/vukem-finances/commit/d0c128a940af9f168ce530f470c4933fae706c73))
- **ci:** Keep the MSIX steps off until MSIX_ENABLED is set ([`951ecbb`](https://github.com/AlexGalhardo/vukem-finances/commit/951ecbb38bfa14021e84546e9455b5c33efb748e))
- **mobile:** Move the app state out of src/app so expo-router reads mobile/app ([`d9af2c2`](https://github.com/AlexGalhardo/vukem-finances/commit/d9af2c252a87ad2bb79a9ef30d24a80ae0afdf58))
- **sync:** Keep the request order of the mutations of one row in a push ([`29c5ce0`](https://github.com/AlexGalhardo/vukem-finances/commit/29c5ce06ca528404d3bcd29e8dc8f448bde94df1))
- **sync:** Register pairing codes in the isolation guard and let any member add a device ([`fcec0d2`](https://github.com/AlexGalhardo/vukem-finances/commit/fcec0d27179f5b73e2e292bb7d92347818e426b0))
- **mobile:** Isolate transactions on the SQLite connection ([`59f3b83`](https://github.com/AlexGalhardo/vukem-finances/commit/59f3b83eaa13b5d01fac2d252b0f44364e28c3d1))
- **sync:** Return real database errors from pairing and fail closed without a peeking limiter ([`6f77080`](https://github.com/AlexGalhardo/vukem-finances/commit/6f77080fc219343811874667aa84c5802e0a276e))
- **mobile:** Keep waiting statements and timed out turns out of open transactions ([`b961d09`](https://github.com/AlexGalhardo/vukem-finances/commit/b961d09b5b80d9d64db5e6388748ae7b423ca857))
- **db:** Run the pairing migrations after the ones published in v0.4.0 ([`634efe0`](https://github.com/AlexGalhardo/vukem-finances/commit/634efe0904a36729eafefbd17c4884de1622561a))
- **desktop:** Keep form errors on their page by sending same-origin referrers ([`6eb52d5`](https://github.com/AlexGalhardo/vukem-finances/commit/6eb52d5e309258e65878d044f91243e3a0052abc))
- **mcp:** Pass resource reads and completions through the grant limit and the audit ([`7713fcd`](https://github.com/AlexGalhardo/vukem-finances/commit/7713fcd1a081710b862dbfe97c3b2906bab1fa54))
- **oauth:** End every AI client grant with the other access of a person ([`b0e034e`](https://github.com/AlexGalhardo/vukem-finances/commit/b0e034ee71b6b8b401f9ca7971cc684cb19c6107))
- **oauth:** Enforce the consent scopes on the server and name clients by their host ([`0ff5f18`](https://github.com/AlexGalhardo/vukem-finances/commit/0ff5f18ad1428c307d1341595231b01870fcb22a))
- **mcp:** Add months with timeutil.AddMonths ([`b35756b`](https://github.com/AlexGalhardo/vukem-finances/commit/b35756b17ff5a093468f0510952ed9f8f39d55c0))
- **i18n:** Save the pt-BR MCP strings as valid UTF-8 ([`a14e070`](https://github.com/AlexGalhardo/vukem-finances/commit/a14e07079465507a862a70ad01dd4f839681b6ea))

### Security

- **api:** Publish the pairing route in OpenAPI and the TypeScript client ([`5b074c8`](https://github.com/AlexGalhardo/vukem-finances/commit/5b074c8d66b9db0ce6afabfaa321dc416eef5e16))

## [0.4.0] - 2026-11-22

### Added

- **telegram:** Read the bot settings ([`6e61f3f`](https://github.com/AlexGalhardo/vukem-finances/commit/6e61f3fc1af215f5c06c8f6d535c80d3cd970c61))
- **telegram:** Add the Bot API client with escaping and token redaction ([`69ff4a3`](https://github.com/AlexGalhardo/vukem-finances/commit/69ff4a3378b57bbb855e2c31ede33280c9f46519))
- **telegram:** Add the Telegram tables and the alert switch ([`2b7c8b4`](https://github.com/AlexGalhardo/vukem-finances/commit/2b7c8b4a576ddfccd704eaba71f0042e82509105))
- **telegram:** Parse an entry from one short message ([`78fc5af`](https://github.com/AlexGalhardo/vukem-finances/commit/78fc5af9901310156df356ae1bc29299d31fccf9))
- **telegram:** Add the bot texts in en, pt-BR and es ([`21e65ab`](https://github.com/AlexGalhardo/vukem-finances/commit/21e65ab8d71fb8cd5934f52dbdb1ab38781017cd))
- **telegram:** Handle updates, link chats and run the bot commands ([`f9698f6`](https://github.com/AlexGalhardo/vukem-finances/commit/f9698f6dacc01b89491ab979e03c7d18aaa35d04))
- **telegram:** Deliver alerts to a linked chat through alerts.Channel ([`4ae05fa`](https://github.com/AlexGalhardo/vukem-finances/commit/4ae05fa3d740cb053204caf9309068f0acef698b))
- **telegram:** Add telegram:set-webhook and telegram:info commands ([`78a308f`](https://github.com/AlexGalhardo/vukem-finances/commit/78a308fc1d5f30fe4de0a20050d75d491388e19a))
- **telegram:** Prune old update ids, codes, drafts and rate windows daily ([`e008512`](https://github.com/AlexGalhardo/vukem-finances/commit/e008512d82f0e5470d308a611aa6d4fbc434bc2d))
- **telegram:** Add the settings page backend with connect, revoke and the alert switch ([`504f73a`](https://github.com/AlexGalhardo/vukem-finances/commit/504f73a8fd5a328a803a29eda168fa437f05e667))
- **telegram:** Add the Telegram settings page and the i18n in three locales ([`67303a4`](https://github.com/AlexGalhardo/vukem-finances/commit/67303a48b9f5c6ec61c691389a611154120b2e53))
- **sync:** Add the sync_devices and sync_ops tables ([`9b0fdbf`](https://github.com/AlexGalhardo/vukem-finances/commit/9b0fdbffb6998a0ff6e16b08148ea98973fd50d0))
- **sync:** Serve the sync protocol under /api/v1/sync ([`a9ba53a`](https://github.com/AlexGalhardo/vukem-finances/commit/a9ba53adfb5cdf0e03ad0356547d98fdfc8b4c17))
- **sync:** Prune the stored results of pushed operations after 30 days ([`9dd9fd0`](https://github.com/AlexGalhardo/vukem-finances/commit/9dd9fd03d6728939a664da5f0b9e8b4c723f4eac))
- **sync:** Add the Devices page ([`2da2c32`](https://github.com/AlexGalhardo/vukem-finances/commit/2da2c3249210dabe4d017e9e360083e42e81bcce))
- **desktop:** Add the loopback gate of the desktop server ([`24184f4`](https://github.com/AlexGalhardo/vukem-finances/commit/24184f499636eca3789d689db1e1e717f46853a2))
- **desktop:** Add the desktop module with paths, key and instance lock ([`20cba1c`](https://github.com/AlexGalhardo/vukem-finances/commit/20cba1c818736a755c51310b27db5da5f75ac60e))
- **desktop:** Start the personal edition on a loopback port ([`06a813a`](https://github.com/AlexGalhardo/vukem-finances/commit/06a813a1801d9dcba83fdd72bf8fd07e3a970eb6))
- **desktop:** Add the headless mode and the smoke script ([`4326bae`](https://github.com/AlexGalhardo/vukem-finances/commit/4326baed84b7e02439c678f89d9445834dd2480c))
- **desktop:** Write a backup zip of the database and the key ([`34e2979`](https://github.com/AlexGalhardo/vukem-finances/commit/34e2979eb6fc348133579d4fa6f50dde3e4e027a))
- **desktop:** Add the native save and open-external routes ([`e679bb8`](https://github.com/AlexGalhardo/vukem-finances/commit/e679bb8e4f5f8903cbf6c48ae3caeaf2f67103c3))
- **desktop:** Open the app in a Wails window with a native menu ([`6535485`](https://github.com/AlexGalhardo/vukem-finances/commit/6535485017e063b8005d8194b705d7201de54754))
- **db:** Add the tables of AI categorization and the assistant ([`74d9dad`](https://github.com/AlexGalhardo/vukem-finances/commit/74d9dad80dc09bcb2cca0d43aacaec1cc9531c09))
- **finance:** Remember the category of a description ([`b63ff35`](https://github.com/AlexGalhardo/vukem-finances/commit/b63ff35b9a4f42da58ec67da8e11d95dec086ca0))
- **ai:** Add the batch categorizer with id and uid allowlists ([`a66f2f4`](https://github.com/AlexGalhardo/vukem-finances/commit/a66f2f428eb1a8e5cd367a5b8d1d3ff2e52954f9))
- **ai:** Record, review and undo category suggestions ([`5fb3d8f`](https://github.com/AlexGalhardo/vukem-finances/commit/5fb3d8ffbb3ea7cecaf9822ab73c390905f82897))
- **ai:** Enqueue and run the ai_categorize job ([`65bfe86`](https://github.com/AlexGalhardo/vukem-finances/commit/65bfe86494b6df3f6db540e1d0313d5134cb5a24))
- **ai:** Read a sentence into a transaction draft ([`0fec63d`](https://github.com/AlexGalhardo/vukem-finances/commit/0fec63dcdeff22ede361c65ec406d72d845868a9))
- **ai:** Add the finance assistant with a validated plan and server figures ([`23e3a4f`](https://github.com/AlexGalhardo/vukem-finances/commit/23e3a4ff063d2fa7b732a96cb6790291d199bb23))
- **ai:** Add the routes of suggestions, drafts and the assistant ([`54a1ba4`](https://github.com/AlexGalhardo/vukem-finances/commit/54a1ba4cfb45e96486c0dd46349077e8419a5f66))
- **ai:** Add the evaluation set and the ai:eval command ([`3ed5501`](https://github.com/AlexGalhardo/vukem-finances/commit/3ed5501d79589615e4efe6f47cbadb395b16c7c0))
- **ui:** Add the suggestions screen, the assistant dialog and the sentence field ([`910bbb4`](https://github.com/AlexGalhardo/vukem-finances/commit/910bbb4982bf2292765ed7343e269f8abe3ad80f))
- **operations:** Add the operation registry, actor, error codes and Invoke ([`6119715`](https://github.com/AlexGalhardo/vukem-finances/commit/611971563179c3c82f1ea902162b1b58d4d46d57))
- **operations:** Replay an idempotency key once per caller ([`f5197fd`](https://github.com/AlexGalhardo/vukem-finances/commit/f5197fd7f437669493e02a6871ec73a63ddc8baf))
- **operations:** Audit every write and destructive call without its data ([`3885016`](https://github.com/AlexGalhardo/vukem-finances/commit/3885016c0eb07d667643cbbb8e1b2853e9536dc8))
- **api:** Add cards, statements, recurrences, goals, reports and export operations ([`5a45d63`](https://github.com/AlexGalhardo/vukem-finances/commit/5a45d63e65012ab752216f1ea5c340b45731530b))
- **operations:** Add the operations:inventory command and its committed output ([`b7dbdfc`](https://github.com/AlexGalhardo/vukem-finances/commit/b7dbdfc032792ad3ca31b0d75b61e3d557dd5abf))

### Changed

- **finance:** Delete and restore synced rows inside a caller transaction ([`030fb61`](https://github.com/AlexGalhardo/vukem-finances/commit/030fb613e1a6b8c54a992b901d5d0f5f420d5626))
- **api:** Run accounts, categories and tags through the operation registry ([`61b3e57`](https://github.com/AlexGalhardo/vukem-finances/commit/61b3e574cdf0de553ec08fccc9d069b4b3c1e440))
- **api:** Run transactions and transfers through the operation registry ([`74dc167`](https://github.com/AlexGalhardo/vukem-finances/commit/74dc1671c7b0da4967b66d1545e58d8612a9af1c))
- **api:** Run budgets and summaries through the operation registry ([`ffdcae2`](https://github.com/AlexGalhardo/vukem-finances/commit/ffdcae236efa417d440c26c1a3f5df2fe3b654d8))

### Fixed

- **telegram:** Keep the undo when recording a saved entry fails ([`bc948a2`](https://github.com/AlexGalhardo/vukem-finances/commit/bc948a240319c77c1b452f15b1c06ed93cc63c0b))
- **telegram:** Move past an update that keeps failing ([`c0f5236`](https://github.com/AlexGalhardo/vukem-finances/commit/c0f5236ed2dddc7ebbb39c4fc8d9dd0edbbdcba7))
- **telegram:** Retry for a minute and tell the chat before skipping an update ([`ce4e52f`](https://github.com/AlexGalhardo/vukem-finances/commit/ce4e52ff3490dafd6c2847eb332a037f1b1eb3bf))
- **sync:** Derive the statement of an entry on the server ([`4276e17`](https://github.com/AlexGalhardo/vukem-finances/commit/4276e17f18f96e0d6324e3638fa8e4610ce54209))
- **sync:** Hold restores and transfer legs to the statement rules ([`eb20b87`](https://github.com/AlexGalhardo/vukem-finances/commit/eb20b87f165f814000bfeb3c9a428009f0db7b8e))
- **desktop:** Keep the gate headers and load the shell script from the server ([`0306599`](https://github.com/AlexGalhardo/vukem-finances/commit/0306599ddcefcdd047e4a9a6e8d8432582edbea1))
- **desktop:** Keep the old file when replacing a download fails ([`d0126ca`](https://github.com/AlexGalhardo/vukem-finances/commit/d0126cadbeeb8b6cc4f4f2d452e17dc058087cb3))
- **ci:** Pin NSIS to a published version ([`0c1e436`](https://github.com/AlexGalhardo/vukem-finances/commit/0c1e43602cdcdb3a84edc4fabc75b1ea881b7c35))
- **ai:** Normalize whitespace before redacting text sent to a provider ([`467f5f7`](https://github.com/AlexGalhardo/vukem-finances/commit/467f5f72bba83ad83aada364e2e200547870162a))
- **ai:** Keep account names out of the sentence to draft payload ([`a8aa352`](https://github.com/AlexGalhardo/vukem-finances/commit/a8aa3525705cb820a048eed84db663099c0f971b))
- **ai:** Send opaque labels for accounts and tags and store only sanitized questions ([`b1ef1cd`](https://github.com/AlexGalhardo/vukem-finances/commit/b1ef1cd5a5493e924e0f45560e5ca18365096024))
- **ai:** Keep the assistant, which spends the AI limit, away from viewers ([`7ca959f`](https://github.com/AlexGalhardo/vukem-finances/commit/7ca959f6c90d99181b29a6505a3d86bc9a2ec07d))
- **ai:** Let a demoted viewer read and erase their own conversations ([`7271271`](https://github.com/AlexGalhardo/vukem-finances/commit/7271271d7173dd850da9f7ed24ca3ebe34fc42ce))
- **operations:** Keep the idempotency claim when the answer cannot be stored ([`8df73bd`](https://github.com/AlexGalhardo/vukem-finances/commit/8df73bdbc959055e8ba74a786662b9b63f1d9525))
- **ci:** Make the test suite and the desktop packaging reliable ([`3f772a2`](https://github.com/AlexGalhardo/vukem-finances/commit/3f772a2422f2f3615f3aa5584822661243413d4c))

## [0.3.0] - 2026-11-22

### Added

- **cards:** Add closing, statement and due date arithmetic ([`c28cdc4`](https://github.com/AlexGalhardo/vukem-finances/commit/c28cdc4b61b9576646f019760a4e1a693247a289))
- **cards:** Add credit_cards table and transaction statement_ref ([`5351780`](https://github.com/AlexGalhardo/vukem-finances/commit/5351780899f36a6499983697e40937b2f2f7a711))
- **cards:** Add card create, update, archive and delete ([`f0a2581`](https://github.com/AlexGalhardo/vukem-finances/commit/f0a258108bff2219b441da4caf6c1e77963228b2))
- **cards:** Tag card purchases with a statement and pay statements ([`61b3280`](https://github.com/AlexGalhardo/vukem-finances/commit/61b32809edf5248eff1023dc62c36381382c7bde))
- **cards:** Add card and statement routes, handlers and pages props ([`3f186b1`](https://github.com/AlexGalhardo/vukem-finances/commit/3f186b1b44f43fda08fa8218a769dcf867dbfc6e))
- **cards:** Add the cards and card statement pages with the g k action ([`7ea31b8`](https://github.com/AlexGalhardo/vukem-finances/commit/7ea31b808538bc3098a839684b6c0a8eb7bd359d))
- **i18n:** Add card translations in three locales and the g k shortcut row ([`f2c3d54`](https://github.com/AlexGalhardo/vukem-finances/commit/f2c3d54693cc4e9fc5d0ea6b218c1748aedfbcc1))
- **cards:** Seed a demo card with two statements ([`79bcc7f`](https://github.com/AlexGalhardo/vukem-finances/commit/79bcc7fb8791f54d7ab31ec6eeb0aafabaa7bdec))
- **api:** Add the token resource column and the vf_ prefix ([`391a3f8`](https://github.com/AlexGalhardo/vukem-finances/commit/391a3f86327fe6b01b0fa94162818b3330498f31))
- **api:** Add scoped token creation, listing and revocation with the Tokens page ([`bd5fdf3`](https://github.com/AlexGalhardo/vukem-finances/commit/bd5fdf3881175fcb1514b277994b10020213320a))
- **api:** Add the bearer middleware chain, me and account endpoints ([`964618f`](https://github.com/AlexGalhardo/vukem-finances/commit/964618f0cb857da9fcc1c9b69f200d3fca24c387))
- **api:** Add category and tag endpoints with client uids and version checks ([`3b40a64`](https://github.com/AlexGalhardo/vukem-finances/commit/3b40a64a28913876ce5d3d3628994dc19a160713))
- **api:** Add transaction and transfer endpoints with filters, pages, client uids and If-Match ([`7ac69d3`](https://github.com/AlexGalhardo/vukem-finances/commit/7ac69d3765cecc3efef211d1b7b233d81d0bdcaf))
- **api:** Add budget and summary endpoints ([`dadb2e0`](https://github.com/AlexGalhardo/vukem-finances/commit/dadb2e09cbe5682f5e6575a7fd73457b85fd56d7))
- **api:** Generate the OpenAPI document and the TypeScript client with freshness tests ([`703253a`](https://github.com/AlexGalhardo/vukem-finances/commit/703253a1fa2427360e45d4bb48ff04f6d262415b))
- **recurrence:** Add the pure recurrence engine ([`1604c1f`](https://github.com/AlexGalhardo/vukem-finances/commit/1604c1fce33291beebfd2b11e29f1005aae641c0))
- **recurrence:** Add the recurrences table and the transaction natural key ([`4060dfa`](https://github.com/AlexGalhardo/vukem-finances/commit/4060dfa463c6fee761172a8c1f2bf69e57858a33))
- **recurrence:** Create, split and delete installment purchases ([`8462732`](https://github.com/AlexGalhardo/vukem-finances/commit/8462732de039bc83df964aae3dee794699c16b34))
- **recurrence:** Generate due occurrences idempotently under the quota lock ([`ec72c6a`](https://github.com/AlexGalhardo/vukem-finances/commit/ec72c6a70a8da8a190a1eddd83e1c60d1f6987a5))
- **recurrence:** Run the generation job hourly ([`60c5060`](https://github.com/AlexGalhardo/vukem-finances/commit/60c50600b10e961800c7227ccdbcd3ed75ed57d1))
- **recurrence:** List what is due in the next days with virtual occurrences ([`e31524b`](https://github.com/AlexGalhardo/vukem-finances/commit/e31524bf8b37dd9ec4337ff52c290716dd1ece2d))
- **recurrence:** Add the recurrence and installment routes and handlers ([`3d2d491`](https://github.com/AlexGalhardo/vukem-finances/commit/3d2d4918a02a35f1bdcc75b5bc1892a0a07a1911))
- **recurrence:** Add the recurrences page, the upcoming list and the installment form ([`e48bd4d`](https://github.com/AlexGalhardo/vukem-finances/commit/e48bd4d04640fab5f570726215d6eb5bf03cbd1d))
- **sync:** Merge a recurrence watermark by the later date ([`218b1ba`](https://github.com/AlexGalhardo/vukem-finances/commit/218b1bad574e5f13903b257c583807bfa87a2dcb))
- **db:** Add budget rollover, goals and notifications tables ([`268874c`](https://github.com/AlexGalhardo/vukem-finances/commit/268874c15e9b4517129bf976ad0ca85e083878ac))
- **budgets:** Sum spending in one grouped query with rollover, parents and an overall budget ([`5550f56`](https://github.com/AlexGalhardo/vukem-finances/commit/5550f564511d0e1129047047320f9cfc525ad00c))
- **goals:** Add goals and contributions with required monthly and projected date ([`cf58a2a`](https://github.com/AlexGalhardo/vukem-finances/commit/cf58a2a75d4c5c601231bb2d21dee566d5405dbf))
- **alerts:** Evaluate alerts from current data in an idempotent job ([`a628f26`](https://github.com/AlexGalhardo/vukem-finances/commit/a628f2605efd8bdc4c997bcb5dd1704a291c4286))
- **alerts:** Add preferences, the mail channel templates and the notification reads ([`e3e4d7c`](https://github.com/AlexGalhardo/vukem-finances/commit/e3e4d7c42544d7b3351f22b9de32c8d2026e234b))
- **budgets:** Add goal, notification and overall budget routes with page props ([`e328735`](https://github.com/AlexGalhardo/vukem-finances/commit/e32873535d01ddfaa8cad642d97d4b5a3fbe9510))
- **i18n:** Add budgets, goals and alerts translations and the goals shortcut ([`7b54651`](https://github.com/AlexGalhardo/vukem-finances/commit/7b546512bbd698f3b8b7d5d60e5262c2b6d88415))
- **ui:** Add the notification bell, goals page and budgets with rollover and overall ([`197a302`](https://github.com/AlexGalhardo/vukem-finances/commit/197a3026d1a2a28504aea3e2f80c08226494fb13))
- **export:** Format cents as a machine decimal ([`c08b855`](https://github.com/AlexGalhardo/vukem-finances/commit/c08b855aff650cfa1c04a4f602d6da501fb0e2ba))
- **reports:** Add the report period and the category index ([`84d4561`](https://github.com/AlexGalhardo/vukem-finances/commit/84d4561c54d448aad815c7e9575af44b7de40f91))
- **reports:** Add cash flow and breakdown by category, account and tag ([`269d58d`](https://github.com/AlexGalhardo/vukem-finances/commit/269d58db8a3cf5da9f0859114c08eeb291cfba44))
- **reports:** Add the net worth series and the period comparison ([`8743f73`](https://github.com/AlexGalhardo/vukem-finances/commit/8743f736630932a86e6886fb2cba08ccdae34dab))
- **export:** Stream the transactions CSV with a formula guard ([`9e686b4`](https://github.com/AlexGalhardo/vukem-finances/commit/9e686b412df143a698dc3ff6e8ee0b54b8d0dc2f))
- **export:** Add the JSON export of the workspace and its schema ([`b6cad12`](https://github.com/AlexGalhardo/vukem-finances/commit/b6cad12526d5c83ced5db1d3f09884eeca0c25c7))
- **export:** Rate limit the downloads and check who may take them ([`634c016`](https://github.com/AlexGalhardo/vukem-finances/commit/634c016159ce4692da1ff9e3869ea6b536690fa8))
- **reports:** Add the reports page, the export menu and the x and g r actions ([`d7f004f`](https://github.com/AlexGalhardo/vukem-finances/commit/d7f004f21af1c1f52c098a31ccf5930c9f09c612))

### Fixed

- **cards:** Move trashed purchases of an emptied statement with the closing day ([`5c5ad5d`](https://github.com/AlexGalhardo/vukem-finances/commit/5c5ad5daea9a8b743f9e4a5cb9a43c4d586c5bf4))
- **api:** Answer unknown paths and methods under /api/v1 with problem+json ([`eca8afb`](https://github.com/AlexGalhardo/vukem-finances/commit/eca8afb9c902d30161929a25b6bd206237b72233))
- **api:** Require tags:write to create a tag through a transaction ([`0b119fd`](https://github.com/AlexGalhardo/vukem-finances/commit/0b119fd385185395f3cdc1d2d8bf992a2e68cb90))
- **billing:** Keep access on a refund of an ended period and after a won dispute ([`936ca87`](https://github.com/AlexGalhardo/vukem-finances/commit/936ca871315a5f09b4f0a06a0ed51c23b08dd777))
- **sync:** Read a recurrence watermark strictly and never past today ([`d38bd5a`](https://github.com/AlexGalhardo/vukem-finances/commit/d38bd5a15e078eedfac115768a761b09355b7535))
- **recurrence:** Bound the engine years and the walk, refuse insane dates ([`af7ad12`](https://github.com/AlexGalhardo/vukem-finances/commit/af7ad128385d3ba0cc9a8934e8dcd985dfae7afa))
- **recurrence:** Post nothing when a future rule moves into the past ([`b3cc416`](https://github.com/AlexGalhardo/vukem-finances/commit/b3cc4169662668d6969b43cd7e293bf0f72ea936))
- **billing:** Decide a refund by the event time and always finish its cancel ([`d1ca96c`](https://github.com/AlexGalhardo/vukem-finances/commit/d1ca96cd364e7469ca2fefc8e93ee8185cbdf058))
- **ui:** Name the notifications page list items so the shared bell prop is not hidden ([`f794244`](https://github.com/AlexGalhardo/vukem-finances/commit/f79424428b2894a7faefda2856555fcc9cf525d8))
- **alerts:** Honor the current mail preference for pending notifications ([`a85ed38`](https://github.com/AlexGalhardo/vukem-finances/commit/a85ed380a4d58d74dbf1a01ad0ca33b63aaab8c9))
- **finance:** Detect formulas behind invisible leading characters ([`b590df7`](https://github.com/AlexGalhardo/vukem-finances/commit/b590df7ddc2faa9dcec88f9c11da98293c82a134))
- **export:** Guard formulas after list separators in CSV cells ([`52bc928`](https://github.com/AlexGalhardo/vukem-finances/commit/52bc928fc3aa4fa1060a937a641568037a4f8a6f))
- **export:** Abort a download that fails after its first byte ([`2853c7f`](https://github.com/AlexGalhardo/vukem-finances/commit/2853c7fed2ae1010d7a2fc644492358e73508513))
- **export:** Guard formulas after line breaks in CSV cells ([`3ffe100`](https://github.com/AlexGalhardo/vukem-finances/commit/3ffe100193b11df474d71320e70312da3f449949))

## [0.2.0] - 2026-11-21

### Added

- **sync:** Add uid, version and tombstone columns with the change log tables ([`d67ddb8`](https://github.com/AlexGalhardo/vukem-finances/commit/d67ddb879953edeef8db96c1c595e0159228ec35))
- **sync:** Stamp uid and version on the synced models ([`059bc2f`](https://github.com/AlexGalhardo/vukem-finances/commit/059bc2f3ceaf20795fbb98fa0e267efc9fca2028))
- **sync:** Add the entity registry and the change-log triggers ([`ccfccc5`](https://github.com/AlexGalhardo/vukem-finances/commit/ccfccc5978286d8ff389ed4aeed992651575be2c))
- **finance:** Soft delete synced rows and add restore ([`8ab1f66`](https://github.com/AlexGalhardo/vukem-finances/commit/8ab1f669f21bf8d263e2065f212296e564f51e87))
- **finance:** Cap money at 2^53 - 1 cents on every write path ([`de9cdc5`](https://github.com/AlexGalhardo/vukem-finances/commit/de9cdc580f980bda7226722ed2e05d29655527b3))
- **sync:** Read the change log by cursor and take a snapshot ([`bf8f0a1`](https://github.com/AlexGalhardo/vukem-finances/commit/bf8f0a13326977cf776b29d79a9e9633afaa31f4))
- **sync:** Merge a mutation into a row by field with last-writer-wins ([`520a17a`](https://github.com/AlexGalhardo/vukem-finances/commit/520a17a48a2c9c9de8c4ba03222fff7344143d11))
- **sync:** Compact tombstones and the change log with a daily job ([`e26450b`](https://github.com/AlexGalhardo/vukem-finances/commit/e26450b5dcce946f3287e3985325706918a219b1))
- **ai:** Add the AI tables, models and the ai.manage ability ([`0e3a667`](https://github.com/AlexGalhardo/vukem-finances/commit/0e3a667e7b5f9281b154880ad7a01283331af05b))
- **ai:** Seal user keys with workspace-bound data and re-seal after rotation ([`a68c849`](https://github.com/AlexGalhardo/vukem-finances/commit/a68c849eda7c48c4e89680bc70abb86a1e2a29b0))
- **ai:** Redact personal data and key-like tokens before text leaves the server ([`3f90877`](https://github.com/AlexGalhardo/vukem-finances/commit/3f90877fef74193e3a2dfddc2b1013bb944469de))
- **ai:** Add the fake provider and the call pipeline with limits and usage ([`4812d43`](https://github.com/AlexGalhardo/vukem-finances/commit/4812d4380ec383c29c27df1555724ce7c90629a9))
- **ai:** Add the anthropic provider over the safe HTTP client ([`1bea8bd`](https://github.com/AlexGalhardo/vukem-finances/commit/1bea8bd31d865e1de63c2126cce5435f5728dcba))
- **ai:** Add the openai and openai-compatible providers and base URL checks ([`596b9cc`](https://github.com/AlexGalhardo/vukem-finances/commit/596b9ccdb7d327353d9de5862ba58c2b6c68b50d))
- **ai:** Add the owner-only routes for the AI key, switches, check and usage ([`b40c268`](https://github.com/AlexGalhardo/vukem-finances/commit/b40c268b8f3ad5b2ba30c80d3d49563d93ca8615))
- **ai:** Add AI translations, the env example entry and agent notes ([`94c9bac`](https://github.com/AlexGalhardo/vukem-finances/commit/94c9bacc72129603402fcd994c6073a815016efa))
- **ai:** Add the AI settings page with the consent dialog and an e2e spec ([`f103dae`](https://github.com/AlexGalhardo/vukem-finances/commit/f103dae986b0ed0a2cf16b7b53ec8e38ed137d02))
- **accounts:** Add archive, position and transfer pairing columns ([`d4ac984`](https://github.com/AlexGalhardo/vukem-finances/commit/d4ac984e9e4cadb6d81005f6af7ed487026d7fb3))
- **accounts:** Validate kinds and add update, archive and unarchive ([`5fb1845`](https://github.com/AlexGalhardo/vukem-finances/commit/5fb1845e3213e7b7dfbb4db78a0beffec9016b8e))
- **accounts:** Derive account balances and net worth ([`41b8b2d`](https://github.com/AlexGalhardo/vukem-finances/commit/41b8b2dbfa6de2a1a9800ac88b630b732693b96f))
- **transfers:** Add two-leg transfers kept out of income, expense and quota ([`cc456b5`](https://github.com/AlexGalhardo/vukem-finances/commit/cc456b537adc888edc47f2a1f401b89518ce0248))
- **accounts:** Add routes for account edit, archive and transfers ([`86111e9`](https://github.com/AlexGalhardo/vukem-finances/commit/86111e99c07ee8a0c8861f0953c4c14f3487ee67))
- **transfers:** Serve the dialog pickers and name the other account of a transfer ([`52e74ee`](https://github.com/AlexGalhardo/vukem-finances/commit/52e74ee7b9da99861794950ad92d865fb3331a60))
- **accounts:** Show balances, net worth and archive controls on the accounts page ([`ebf4387`](https://github.com/AlexGalhardo/vukem-finances/commit/ebf4387c7b121f7a7b18f9531375247e7c7e590b))
- **accounts:** Seed a second account and a transfer in the demo data ([`0b80f10`](https://github.com/AlexGalhardo/vukem-finances/commit/0b80f10c61db19c7a20ae7dcf634ed19eb271708))
- **transfers:** Add quick add dialogs for expense, income and transfer ([`55f7533`](https://github.com/AlexGalhardo/vukem-finances/commit/55f7533cf2982babce24a62fc8782da90468bfd9))
- **billing:** Add revocation columns, PIX tables and billing events ([`cf9822f`](https://github.com/AlexGalhardo/vukem-finances/commit/cf9822f6f8e31cd4a03889af8e08724f9ac399f5))
- **billing:** Derive the plan from one Entitled function ([`085d84b`](https://github.com/AlexGalhardo/vukem-finances/commit/085d84b02707b5d9ea86b7a41918970220ddaa5a))
- **billing:** Revoke Pro on a full refund and cancel the subscription ([`2f151e6`](https://github.com/AlexGalhardo/vukem-finances/commit/2f151e6a112437f8d65045187524904bbc821a97))
- **billing:** Add the AbacatePay client and PIX settings ([`2710b36`](https://github.com/AlexGalhardo/vukem-finances/commit/2710b36cae55bf9fd7b9aef566a11959c8416662))
- **billing:** Add PIX checkout with idempotent activation and stacked months ([`eee1b82`](https://github.com/AlexGalhardo/vukem-finances/commit/eee1b82c70f14cfdbae50dc530ed07144c397904))
- **billing:** Add the AbacatePay webhook with secret, signature and re-check ([`620c255`](https://github.com/AlexGalhardo/vukem-finances/commit/620c255755221edf854370416ec9e7864da010ae))
- **billing:** Reconcile PIX payments, revoke refunds and add the admin action ([`604e6e1`](https://github.com/AlexGalhardo/vukem-finances/commit/604e6e16b2835af2776df38785fc9dfc9f035e00))
- **billing:** Add the PIX start and poll routes with limits and props ([`24d9df8`](https://github.com/AlexGalhardo/vukem-finances/commit/24d9df8c2639c54a93c9621b8de00a5e71137172))
- **billing:** Build the billing page with the PIX dialog and revocation reasons ([`ab2039f`](https://github.com/AlexGalhardo/vukem-finances/commit/ab2039f3b1cafa5fd997b03d6ba04c5ab06e203d))
- **billing:** Count PIX in admin revenue and add the refunded and receipt mails ([`dee3039`](https://github.com/AlexGalhardo/vukem-finances/commit/dee3039ea86438642bc0bcd1e38b23b5be879cf9))
- **db:** Add category tree columns, tags and the import hash ([`dfd7564`](https://github.com/AlexGalhardo/vukem-finances/commit/dfd756410834dbc678bb3f1b29526820a739ddb1))
- **categories:** Update, archive and delete with reassignment ([`a37308b`](https://github.com/AlexGalhardo/vukem-finances/commit/a37308bee36e81ab3138a97017790baa879ec839))
- **tags:** Create inline, limit, attach and detach ([`a059e71`](https://github.com/AlexGalhardo/vukem-finances/commit/a059e71bec39984d4f701fc20bd7122e8522e0d4))
- **finance:** Search and filter transactions with pages ([`452bbf8`](https://github.com/AlexGalhardo/vukem-finances/commit/452bbf827b5e438be3288b74f49aeb8316f75a9b))
- **categories:** Routes for categories, tags and the filtered list ([`b8a1971`](https://github.com/AlexGalhardo/vukem-finances/commit/b8a197102543fe1108158596e205e9ded69ef75f))
- **import:** Add the OFX parser with hostile-file fixtures ([`823898c`](https://github.com/AlexGalhardo/vukem-finances/commit/823898cb7aaccd8ac4dec3ebfac3d44ca126301e))
- **import:** Detect duplicates, preview totals and store a report ([`267f605`](https://github.com/AlexGalhardo/vukem-finances/commit/267f60565e57de1a9aaa76fd52e7645f4f180645))
- **i18n:** Add categories, search and import strings in three locales ([`1b2cec3`](https://github.com/AlexGalhardo/vukem-finances/commit/1b2cec313f64e78214b9758ec2a0cb69ed0e2ec7))
- **ui:** Categories tree, tags, filters, search action and import flow ([`a8c844c`](https://github.com/AlexGalhardo/vukem-finances/commit/a8c844c4394e50c9923f279369e7234fcba2cec4))

### Changed

- **finance:** Split models, routes and handlers by domain ([`c5e6bb4`](https://github.com/AlexGalhardo/vukem-finances/commit/c5e6bb481cdea1bb0887c8206457c9928ee1e5f5))
- **import:** Read statements through StatementParser ([`b839052`](https://github.com/AlexGalhardo/vukem-finances/commit/b839052200905a52cfdc176d7bce4d6596e6ae00))

### Fixed

- **sync:** Read ISO timestamps when backfilling uids and stamp the log the same way ([`ddb5bce`](https://github.com/AlexGalhardo/vukem-finances/commit/ddb5bce6f04c0d776eaab698d9be06812196c253))
- **sync:** Keep the cursor at the compacted seq so a snapshot cannot loop ([`b806273`](https://github.com/AlexGalhardo/vukem-finances/commit/b806273ffc67a84905eb330f091ac01069211461))
- **ai:** Use the current Claude model and treat refused or cut-off answers as errors ([`de4ce49`](https://github.com/AlexGalhardo/vukem-finances/commit/de4ce4941a855b396b5cefc75c63d18fed9753bc))
- **ai:** Close the usage reservation when the client disconnects ([`d7c43f0`](https://github.com/AlexGalhardo/vukem-finances/commit/d7c43f0dcb7882c18e7d16327166e7408c3df689))
- **ai:** Keep billed and unknown-outcome calls counted against the monthly limit ([`d42d713`](https://github.com/AlexGalhardo/vukem-finances/commit/d42d713b63c3da8a8a27d1c29684ef78fe435257))
- **accounts:** Refuse new entries on archived accounts ([`de46c10`](https://github.com/AlexGalhardo/vukem-finances/commit/de46c10e19b0ba2e9c5472022db4b93fd4477409))
- **docker:** Cross-compile the image instead of emulating arm64 ([`07e3f43`](https://github.com/AlexGalhardo/vukem-finances/commit/07e3f434f0dec006b494a25130ed4b022bad35f2))
- **billing:** Read the plan inside the transaction that checks the quota ([`39b0a44`](https://github.com/AlexGalhardo/vukem-finances/commit/39b0a446ee3843082bfce26effa6319d919e229d))
- **billing:** Keep the current PIX period when an ended one is refunded ([`5899fb4`](https://github.com/AlexGalhardo/vukem-finances/commit/5899fb4c2ac9d60bed7240ab808f741a1a233772))
- **billing:** Give back only the months a PIX dispute withheld ([`8b230f8`](https://github.com/AlexGalhardo/vukem-finances/commit/8b230f85379d6f641ea2a351d90f58216be4dd82))
- **billing:** Restore a won PIX dispute on its original term ([`d287704`](https://github.com/AlexGalhardo/vukem-finances/commit/d287704b1cd584d84a2f51d61cb37f4c9d48138a))
- **billing:** Recompute PIX months from paid payments on a won dispute ([`e4b34a5`](https://github.com/AlexGalhardo/vukem-finances/commit/e4b34a5b658b3ca2ba86ab214770fda79b7b8201))
- **billing:** Keep PIX paid months exact across a lapsed period ([`f5d13ed`](https://github.com/AlexGalhardo/vukem-finances/commit/f5d13ed5d67eff94247ea3758bbb62258992a15e))
- **finance:** Refuse amounts with malformed separators or a second number ([`aa63a36`](https://github.com/AlexGalhardo/vukem-finances/commit/aa63a36c8169524638ec4f0e99586bdeedf20f52))
- **finance:** Keep restored transactions off deleted categories and tags ([`e47b1eb`](https://github.com/AlexGalhardo/vukem-finances/commit/e47b1eb05bbcd27f829c783043e86a0ffd979a93))

## [0.1.0] - 2026-11-21

### Added

- **edition:** Add APP_EDITION loading ([`f22d591`](https://github.com/AlexGalhardo/vukem-finances/commit/f22d591d572f91d2abbeff1409e8969227c6a574))
- **edition:** Stop boot on an invalid APP_EDITION ([`1096ec1`](https://github.com/AlexGalhardo/vukem-finances/commit/1096ec1fe5a88efad64baada1ba1c2e17359e074))
- **edition:** Lift every plan cap in the personal edition ([`92a4a73`](https://github.com/AlexGalhardo/vukem-finances/commit/92a4a7378430968f8dd630e943cb54fa274cb90f))
- **edition:** Leave billing and revenue out of the personal edition ([`c57b698`](https://github.com/AlexGalhardo/vukem-finances/commit/c57b698916660b1c90890523896c745e9765b257))
- **edition:** Close public sign-up and add the setup flow in personal ([`f208dd5`](https://github.com/AlexGalhardo/vukem-finances/commit/f208dd5aed8dc6c2bb13d111d02a211745d3f89e))
- **edition:** Register through an invite link in personal ([`c82e06d`](https://github.com/AlexGalhardo/vukem-finances/commit/c82e06dd7c77b355156be852446a26b97add1b90))
- **edition:** Make social login find-only in personal ([`8796b08`](https://github.com/AlexGalhardo/vukem-finances/commit/8796b0852a3552d03a433ce98443236db2c01a8c))
- **edition:** Keep a single implicit workspace in personal ([`1127c94`](https://github.com/AlexGalhardo/vukem-finances/commit/1127c942c1675dea2c50cc64d792c3e642b65d2b))
- **edition:** Hide the landing, switcher and billing entry in personal ([`f9fd710`](https://github.com/AlexGalhardo/vukem-finances/commit/f9fd71004ca83bd2aa6bf59105d6e7b5b7351023))
- **docker:** Stamp the release version in the image and show it in about ([`ddfb6b5`](https://github.com/AlexGalhardo/vukem-finances/commit/ddfb6b5343b8c4f5eefe6353c9c6a5be95f9e825))
- **docker:** Add the VPS stack with Caddy and automatic HTTPS ([`417bb52`](https://github.com/AlexGalhardo/vukem-finances/commit/417bb52d3552d74afcf25aeb102aae906377cedc))
- **cli:** Add user:reset-password for a locked-out owner ([`c9436d1`](https://github.com/AlexGalhardo/vukem-finances/commit/c9436d1f216139062c5055eab3ec1ed8ca284768))
- **docker:** Let compose.prod.yaml run a published image through APP_IMAGE ([`3977817`](https://github.com/AlexGalhardo/vukem-finances/commit/3977817abbde5fc4d82a0d131c0e1d43f242f612))
- **workspace:** Add the read-only viewer role ([`1a159bd`](https://github.com/AlexGalhardo/vukem-finances/commit/1a159bdc25dc131025f87fc42cdc11917f4259d9))
- **workspace:** Change roles, transfer ownership and leave ([`c0bbc1e`](https://github.com/AlexGalhardo/vukem-finances/commit/c0bbc1e6d91944818ccf192952195be6d354eb50))
- **workspace:** Add a validated timezone to each workspace ([`f60d96e`](https://github.com/AlexGalhardo/vukem-finances/commit/f60d96e4c89f56cd4a702a5891af35a3e2644f38))
- **finance:** Send the monthly summary in each workspace's local time ([`3aad3ae`](https://github.com/AlexGalhardo/vukem-finances/commit/3aad3ae03bfe26abfb633940734b401e5f04a13e))
- **workspace:** Rate limit the workspace routes ([`be2ed9b`](https://github.com/AlexGalhardo/vukem-finances/commit/be2ed9b23072ffed24e03a4d8bf490b764b8b014))
- **workspace:** Record sensitive workspace actions in an audit log ([`ccd9886`](https://github.com/AlexGalhardo/vukem-finances/commit/ccd98861561383be11a333ceb3ea56f3194a39f1))
- **ui:** Add design tokens with light and dark themes and a contrast test ([`0a51699`](https://github.com/AlexGalhardo/vukem-finances/commit/0a51699485931da5f1dac7502ee7282b59a5e155))
- **ui:** Add the shortcut registry, key matcher, typing guard and reserved list ([`0ad9699`](https://github.com/AlexGalhardo/vukem-finances/commit/0ad9699375dd220d855a3ec91f6e94ef37debb0f))
- **ui:** Add PageHeader, MoneyText, DataTable, Tabs, Pagination, Toast and Combobox ([`575fcc9`](https://github.com/AlexGalhardo/vukem-finances/commit/575fcc9230c2b3d0336b7dfc215650fff3dc8ef2))
- **ui:** Add the command palette, the shortcut help and the navigation actions ([`afee8a9`](https://github.com/AlexGalhardo/vukem-finances/commit/afee8a9351ecf4d836f2e683afa46881a87e47ff))
- **ui:** Replace AppLayout with the Shell, a skip link and focus on navigation ([`815dbce`](https://github.com/AlexGalhardo/vukem-finances/commit/815dbceea16a2435575352746ce4ae421a4dbf1d))

### Changed

- **i18n:** Split the dictionaries per locale and domain with a parity test ([`f21b7f3`](https://github.com/AlexGalhardo/vukem-finances/commit/f21b7f3224da6820e52abb047ce36b96014ce1fe))
- **ui:** Use PageHeader on the signed-in pages and format the code with Biome ([`739615d`](https://github.com/AlexGalhardo/vukem-finances/commit/739615df22b0fd07d38a58d765b9226f125e7240))

### Fixed

- **accounts:** Fail closed on email verification in saas ([`52c110c`](https://github.com/AlexGalhardo/vukem-finances/commit/52c110c8dee77c0b23e2cf1447e389b47036f075))
- **edition:** Answer 404 on /admin/revenue in the personal edition ([`da29434`](https://github.com/AlexGalhardo/vukem-finances/commit/da2943415ca0a4369e88c7bbfb38f12bdc773476))
- **docker:** Create the backups and restore directories in the storage volume ([`e1c016d`](https://github.com/AlexGalhardo/vukem-finances/commit/e1c016d41662078e542fd57f9378505b7e870018))
- **finance:** Read today and the month in the workspace timezone ([`30ecd2f`](https://github.com/AlexGalhardo/vukem-finances/commit/30ecd2f65c0e269ea3dc79a41adfdaa160dcb260))
- **finance:** Clamp month arithmetic to the last day of the month ([`5b6018f`](https://github.com/AlexGalhardo/vukem-finances/commit/5b6018fc4df012ff3c7ca830a2cea4180c37d908))
- **security:** Limit failed two-factor codes per account ([`61ff14c`](https://github.com/AlexGalhardo/vukem-finances/commit/61ff14c2c9cf37aa72f5822c50ca7eb03016f9b8))
- **security:** Hold the two-factor budget under concurrent requests ([`c19b374`](https://github.com/AlexGalhardo/vukem-finances/commit/c19b37488e0037d58efd0e1997a551b2375a154b))
- **finance:** Keep sending monthly summaries after one workspace fails ([`7cee804`](https://github.com/AlexGalhardo/vukem-finances/commit/7cee8046b73ce7fc3be38681f4cf5bec8a9af921))
- **ui:** Return focus to the opener, focus the heading and mark only the active nav item ([`dcf0c47`](https://github.com/AlexGalhardo/vukem-finances/commit/dcf0c473aaa06e38f9a651a2707f9a3cf37c21ff))
- **ci:** Pass the config to the secret scan ([`6027e97`](https://github.com/AlexGalhardo/vukem-finances/commit/6027e972503cfd5eea99a49720d7febb7f4c1a23))
- **release:** Keep the generated changelog free of extra blank lines ([`5a206eb`](https://github.com/AlexGalhardo/vukem-finances/commit/5a206eb70cff7495782e6f39e2e1ca07da605d72))

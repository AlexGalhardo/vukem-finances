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

## [0.8.0] - 2026-11-23

### Fixed

- **release:** Create the release tag as an annotated tag ([`959300b`](https://github.com/AlexGalhardo/vukem-finances/commit/959300b7211dc29e31fe323c884cc8c42056bfd4))
- **scaffold:** Keep the QA inventory and the site data out of the export ([`db85b34`](https://github.com/AlexGalhardo/vukem-finances/commit/db85b34ecedd8815b880edc12cd9bd8724cce1a2))

## [0.7.0] - 2026-11-23

### Added

- **site:** Add plans:export with a -check mode wired into CI ([`c74d0b4`](https://github.com/AlexGalhardo/vukem-finances/commit/c74d0b44725379882fcb17b7ba72c7eb13f8f6a6))
- **landing:** Remove the Inertia landing and redirect / in both editions ([`9c5f885`](https://github.com/AlexGalhardo/vukem-finances/commit/9c5f8851fd8f6029e3d72f2c6b605e7dac5588be))
- **site:** Add static docs search with pagefind 1.6.0 ([`c6f2b96`](https://github.com/AlexGalhardo/vukem-finances/commit/c6f2b965c758fcce3e6bd5711ca3bb6ae006843b))
- **db:** Keep a monthly summary of the transactions with triggers ([`52652c7`](https://github.com/AlexGalhardo/vukem-finances/commit/52652c7c77c021ed61145476d4e4daf362700f72))
- **workspace:** Let an installation set the read limit of a user ([`4b20e16`](https://github.com/AlexGalhardo/vukem-finances/commit/4b20e16fe33695ab1f7eec19152672b57ed4aea6))
- **scaffold:** Export the app as the Vukem finance scaffold ([`f63fa89`](https://github.com/AlexGalhardo/vukem-finances/commit/f63fa89752632748e2a32f79b75bbc4c9be1c102))

### Changed

- **finance:** Read balances, dashboard and budget spending from the monthly summary ([`7624db3`](https://github.com/AlexGalhardo/vukem-finances/commit/7624db3f344aa94de79767926cd4b19f5aa38a40))
- **reports:** Read cash flow, categories, accounts, totals and net worth from the monthly summary ([`4f26a49`](https://github.com/AlexGalhardo/vukem-finances/commit/4f26a4970c8f8f747aef12d43059ac9fc12c1104))
- **export:** Read plain columns and continue a chunk with a row value comparison ([`1591d01`](https://github.com/AlexGalhardo/vukem-finances/commit/1591d01c440664051b104653a3843695463fbc1a))

### Fixed

- **release:** Relink the changelog to main after the rebase-merge ([`0d6b670`](https://github.com/AlexGalhardo/vukem-finances/commit/0d6b67070ff0e3b929df14cd6e46642dcc516238))
- **release:** Regenerate the site changelog in the release commit ([`79bfdd1`](https://github.com/AlexGalhardo/vukem-finances/commit/79bfdd1a04e85efad4afb024181fa335ae29ec52))
- **site:** Mark the Docs link of the header as current on docs pages ([`173b2f5`](https://github.com/AlexGalhardo/vukem-finances/commit/173b2f5839ffbf3ace4075c219e1d5cce879f908))
- **privacy:** Redact the address of an invitation when the person deletes their account ([`7843a3f`](https://github.com/AlexGalhardo/vukem-finances/commit/7843a3fa57c19d7cf48db4daaf7892a91d91ac5b))
- **privacy:** End the sessions of an account again before its purge ([`49bf8a2`](https://github.com/AlexGalhardo/vukem-finances/commit/49bf8a2ad0373c445d4a12774444ffdabfe54383))
- **ci:** Write the cron expression of the inventory as code so the Markdown lint passes ([`b612494`](https://github.com/AlexGalhardo/vukem-finances/commit/b6124947387d16ade35a50d48f646edb4f7b0a9d))
- **ui:** Close the accessibility findings of the page audit ([`291cf1d`](https://github.com/AlexGalhardo/vukem-finances/commit/291cf1d2a48c58ff2e3c6a75eb9e8cea3874b58b))
- **ui:** Accept the suggestion that holds the focus ([`c035af0`](https://github.com/AlexGalhardo/vukem-finances/commit/c035af0ec6c2563f9f56c6a44e8e62284e818ab6))
- **ai:** Make the per-minute call limit configurable ([`c033e44`](https://github.com/AlexGalhardo/vukem-finances/commit/c033e44017aa589d73d942ab9e9502f4772650e7))
- **ci:** Keep Playwright output out of the Markdown lint ([`d5eb9a7`](https://github.com/AlexGalhardo/vukem-finances/commit/d5eb9a7743b2ca5044807ba5c6527676b52c9baa))

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

- **cli:** Generate commands and flags from the operation registry ([`81f9cdc`](https://github.com/AlexGalhardo/vukem-finances/commit/81f9cdc24a949cc0aa779af43bd92c52ef3fb757))
- **cli:** Add local mode through Invoke, refused outside the personal edition ([`315b625`](https://github.com/AlexGalhardo/vukem-finances/commit/315b625fa53c77ade1ffe4c7a612200412952e72))
- **cli:** Add remote mode with login, logout and token storage ([`238a984`](https://github.com/AlexGalhardo/vukem-finances/commit/238a9841b41cc974e33a7bf3d2a0932c04955377))
- **cli:** Add table, JSON and CSV output with exact amounts ([`b5f0b16`](https://github.com/AlexGalhardo/vukem-finances/commit/b5f0b16ed81aa608847a3102b8a21608212164e6))
- **cli:** Confirm destructive commands and require --yes without a terminal ([`e217cab`](https://github.com/AlexGalhardo/vukem-finances/commit/e217cab032534cc9bbaec1124361817e76cb0637))
- **cli:** Add bash, zsh, fish and PowerShell completion ([`eb35806`](https://github.com/AlexGalhardo/vukem-finances/commit/eb35806673bc24a8128422823904b21a6f3a4aaa))
- **cli:** Generate the CLI reference from the registry ([`68e7e41`](https://github.com/AlexGalhardo/vukem-finances/commit/68e7e4127f78b239ba3f4856a9671e1bcec9538f))
- **desktop:** Ship a macOS disk image and fail the release upload when an installer is missing ([`6152638`](https://github.com/AlexGalhardo/vukem-finances/commit/61526380062fee84321c24daa99854e412337ee9))
- **desktop:** Package the Windows app as an MSIX ([`7c4c767`](https://github.com/AlexGalhardo/vukem-finances/commit/7c4c7675a188bca706f35ce272c4dfcaa430fe4b))
- **ci:** Test the MSIX on every run and submit it to the Microsoft Store on a release ([`749893c`](https://github.com/AlexGalhardo/vukem-finances/commit/749893ce26e6d51df0e31c002bfcbc0cf0396021))
- **mobile:** Scaffold the Expo app with pinned SDK 58 packages ([`0acb2cd`](https://github.com/AlexGalhardo/vukem-finances/commit/0acb2cd04b3a50d239cc2dfe042c8aadabe0b48c))
- **mobile:** Port parseAmount, formatMoney, newUid and addMonths ([`6ac6f72`](https://github.com/AlexGalhardo/vukem-finances/commit/6ac6f72c895753283d4ca80c0b7efb3131e18ef2))
- **mobile:** Add the local SQLite schema, migrations and outbox repositories ([`75370e1`](https://github.com/AlexGalhardo/vukem-finances/commit/75370e17f30d735d1b3797e300eaca8cda5dcff1))
- **mobile:** Add balances, transfers and category rules over the repositories ([`e658c78`](https://github.com/AlexGalhardo/vukem-finances/commit/e658c78d52c3b39caee2d4ce6471550d3481e11d))
- **mobile:** Add budgets, cash flow and category reports ([`93b8eb1`](https://github.com/AlexGalhardo/vukem-finances/commit/93b8eb16b3ba1aae531fbc47ed157ec2532f67cf))
- **mobile:** Add the recurrence engine, generation and installments ([`837321a`](https://github.com/AlexGalhardo/vukem-finances/commit/837321a8fe5043cd0f537d84a1a2b7505b98fc52))
- **mobile:** Add card statements and export their vectors from Go ([`e682b35`](https://github.com/AlexGalhardo/vukem-finances/commit/e682b35c1dd44eb3d975b0e403d4156fac6ffb8a))
- **mobile:** Generate theme tokens from the CSS and add the en, pt-BR and es catalogs ([`13625eb`](https://github.com/AlexGalhardo/vukem-finances/commit/13625ebe0a4485c01ace5489b3b9b59c3156956b))
- **mobile:** Add the shared components with role and label tests ([`ee3783b`](https://github.com/AlexGalhardo/vukem-finances/commit/ee3783b5a3a19e042f88631ae2f6f94c05b44ce6))
- **mobile:** Add the first run, home, add and transactions screens ([`9d9b0df`](https://github.com/AlexGalhardo/vukem-finances/commit/9d9b0dff28d9536436769e2a9b19317cf24acadc))
- **mobile:** Add the accounts, cards, categories, tags, budgets, recurrences and reports screens ([`099430f`](https://github.com/AlexGalhardo/vukem-finances/commit/099430f503a96ef46aaddc0ae9ac9f8a7967981c))
- **mobile:** Add settings, the app lock, the local JSON backup and the no-network guard ([`cb97a0c`](https://github.com/AlexGalhardo/vukem-finances/commit/cb97a0c7674d40856d15b1cfda4a2318107380f2))
- **sync:** Pair a phone with a short single-use code ([`7c78242`](https://github.com/AlexGalhardo/vukem-finances/commit/7c782422848d97dd93d94d7d18b42755ac58fc00))
- **sync:** Add the HTTP transport, the address rules and secure token storage ([`fae22fa`](https://github.com/AlexGalhardo/vukem-finances/commit/fae22fa41a4fdab1eb8ed08dc3cf1c63fcd002b8))
- **sync:** Add the outbox batches, the sync state and the sync tables ([`7d0511d`](https://github.com/AlexGalhardo/vukem-finances/commit/7d0511d9305efb8e828a383c9d464d9da9045888))
- **sync:** Apply pulled pages with pending-edit protection and an atomic cursor ([`56a7b96`](https://github.com/AlexGalhardo/vukem-finances/commit/56a7b9621e062428adbcb6a0ff7b24c70e82593f))
- **sync:** Handle push results, re-keying duplicates and rejections ([`8e4af14`](https://github.com/AlexGalhardo/vukem-finances/commit/8e4af14a3ed38e1c57fff1a8f5ee08379771b740))
- **sync:** Add the integrity pass and the resync that keeps dirty rows ([`c06b707`](https://github.com/AlexGalhardo/vukem-finances/commit/c06b70774f2665eb228d7e9e385f614a9d2f6493))
- **sync:** Add the sync engine with states, triggers, a mutex and backoff ([`0f6c422`](https://github.com/AlexGalhardo/vukem-finances/commit/0f6c4224d55ff69ce1dc04bdb312e0c6d47877c2))
- **sync:** Add connect and the first-connection merge choice with name adoption ([`47f1dbd`](https://github.com/AlexGalhardo/vukem-finances/commit/47f1dbd295b5ea204dc88b4bdafe53511b0c7736))
- **sync:** Add the sync strings in en, pt-BR and es ([`955bedc`](https://github.com/AlexGalhardo/vukem-finances/commit/955bedc45319bb8b6e67b33384b45a3fe12a6d31))
- **sync:** Add the Connect, merge choice, status, conflicts and problems screens ([`023b2d6`](https://github.com/AlexGalhardo/vukem-finances/commit/023b2d6dc82b8f13ed5f584e40acee25c77f1cbe))
- **operations:** Offer the registry as MCP tools ([`1a80f78`](https://github.com/AlexGalhardo/vukem-finances/commit/1a80f789ec37a2d4a4ff03b6a015372dc9ebe2a6))
- **mcp:** Read MCP_TOOLSETS and MCP_ALLOW_DESTRUCTIVE ([`9c682e7`](https://github.com/AlexGalhardo/vukem-finances/commit/9c682e7ac1040a6f25b683a0096adbe02d4f784d))
- **mcp:** Serve the finances over MCP on stdio ([`158e71b`](https://github.com/AlexGalhardo/vukem-finances/commit/158e71b16af430563e95c9b6412b31d826300590))
- **mcp:** Report progress and stop a cancelled call through its context ([`3a9625d`](https://github.com/AlexGalhardo/vukem-finances/commit/3a9625d0df287e4492942b389a3cc78a24f7a177))
- **mcp:** Add MCP to the surface parity test and generate its reference ([`97c50b3`](https://github.com/AlexGalhardo/vukem-finances/commit/97c50b3d77cb85bffe38c5eec1e84a71fbfde38f))
- **oauth:** Add the grant binding table and the framework oauth tables ([`1b40cd9`](https://github.com/AlexGalhardo/vukem-finances/commit/1b40cd99db1fed4b39ecabeed82ae191e9cbc095))
- **mcp:** Let an adapter limit and audit each tool call ([`f971e3f`](https://github.com/AlexGalhardo/vukem-finances/commit/f971e3f84c02213873398531c6a03df8589e5c11))
- **oauth:** Wire the Vukem authorization server to workspaces ([`21b3621`](https://github.com/AlexGalhardo/vukem-finances/commit/21b362172aa813205e69d7dace375ce4c7ae7710))
- **mcp:** Serve the tools over Streamable HTTP behind OAuth ([`51ad12c`](https://github.com/AlexGalhardo/vukem-finances/commit/51ad12c14d0e5fa788c1c482ed4db4c14e0b59c4))
- **oauth:** Serve the metadata, authorization and token endpoints and /mcp ([`8ebeb4a`](https://github.com/AlexGalhardo/vukem-finances/commit/8ebeb4add8947cdf933c428e4682e724c5f6d5fa))
- **edition:** Make the personal edition offline-first ([`745a23e`](https://github.com/AlexGalhardo/vukem-finances/commit/745a23e75effe1c2946d89f9979357292d2fc43a))

### Fixed

- **workspace:** Install the mail suppression check once per mailer, so concurrent jobs no longer race on it
- **cli:** Refuse a download to a terminal and write --out atomically ([`33cbdc3`](https://github.com/AlexGalhardo/vukem-finances/commit/33cbdc366d4a048296cb7e32e526147d3876ff75))
- **cli:** Harden CSV and table output against terminal and spreadsheet injection ([`bcf3674`](https://github.com/AlexGalhardo/vukem-finances/commit/bcf36742c838c06e71510916449f0ffe83e880be))
- **cli:** Guard every ;- and tab-separated segment of a CSV cell ([`07ae3a5`](https://github.com/AlexGalhardo/vukem-finances/commit/07ae3a534df1bbd94d043d5029838401df97efce))
- **cli:** Quote every CSV text cell so no reader delimiter can split a formula out ([`595fc45`](https://github.com/AlexGalhardo/vukem-finances/commit/595fc4543a78d056a05688bbfe86da8da82e01ea))
- **cli:** Guard every CSV segment and stream the output ([`59e0645`](https://github.com/AlexGalhardo/vukem-finances/commit/59e0645400889695b5dedd01425700b15f9d21aa))
- **cli:** Never send an environment token to the stored server ([`1b9be8f`](https://github.com/AlexGalhardo/vukem-finances/commit/1b9be8fb9deb0d401257424abea786204eb896a4))
- **cli:** Guard comma segments of multiline CSV cells ([`38144fc`](https://github.com/AlexGalhardo/vukem-finances/commit/38144fc6e7966cd4f26e7e3ee6b4b9a38c369de9))
- **cli:** Treat every separator and line break as a CSV guard boundary ([`9942f85`](https://github.com/AlexGalhardo/vukem-finances/commit/9942f85792ed701da68fcf36f9a9c384e2330835))
- **ci:** Keep the MSIX steps off until MSIX_ENABLED is set ([`fbdf862`](https://github.com/AlexGalhardo/vukem-finances/commit/fbdf862ea077c190d563c576e67c64b7ff012333))
- **mobile:** Move the app state out of src/app so expo-router reads mobile/app ([`6435453`](https://github.com/AlexGalhardo/vukem-finances/commit/6435453d838418bac91520712eacae800e84d5b9))
- **sync:** Keep the request order of the mutations of one row in a push ([`4f4af6e`](https://github.com/AlexGalhardo/vukem-finances/commit/4f4af6e6ecf116b586644cf9e626e2c6f2966d06))
- **sync:** Register pairing codes in the isolation guard and let any member add a device ([`42e0716`](https://github.com/AlexGalhardo/vukem-finances/commit/42e071668aa3221c00105c47d59238426424d2a8))
- **mobile:** Isolate transactions on the SQLite connection ([`afe84d8`](https://github.com/AlexGalhardo/vukem-finances/commit/afe84d89721c73a5f1e611e014a726248aa06123))
- **sync:** Return real database errors from pairing and fail closed without a peeking limiter ([`8370140`](https://github.com/AlexGalhardo/vukem-finances/commit/837014046006f3d8804154e0b422e70af82be372))
- **mobile:** Keep waiting statements and timed out turns out of open transactions ([`1309610`](https://github.com/AlexGalhardo/vukem-finances/commit/13096101fc89bcbf3982c8dcb8726a064e6ac4d0))
- **db:** Run the pairing migrations after the ones published in v0.4.0 ([`148bd86`](https://github.com/AlexGalhardo/vukem-finances/commit/148bd86cf7d0166b3e44de560582f592e913f259))
- **desktop:** Keep form errors on their page by sending same-origin referrers ([`ac4d2d4`](https://github.com/AlexGalhardo/vukem-finances/commit/ac4d2d4b29737a33daeeb93151325522dc61d0ff))
- **mcp:** Pass resource reads and completions through the grant limit and the audit ([`59c09df`](https://github.com/AlexGalhardo/vukem-finances/commit/59c09df5fbf87c57ff05571a633d90b89a1ed19f))
- **oauth:** End every AI client grant with the other access of a person ([`97436e5`](https://github.com/AlexGalhardo/vukem-finances/commit/97436e5e7f036dd36d887d44360c470b6118a1a1))
- **oauth:** Enforce the consent scopes on the server and name clients by their host ([`56607e6`](https://github.com/AlexGalhardo/vukem-finances/commit/56607e6014a728469660143932c7c008d842585e))
- **mcp:** Add months with timeutil.AddMonths ([`a7ebdab`](https://github.com/AlexGalhardo/vukem-finances/commit/a7ebdabd872d7e5933bc19b12e59d3440cb3082b))
- **i18n:** Save the pt-BR MCP strings as valid UTF-8 ([`5d177ee`](https://github.com/AlexGalhardo/vukem-finances/commit/5d177ee5c4f7c38dcce5ea501c05ca9d91cdc6d7))

### Security

- **api:** Publish the pairing route in OpenAPI and the TypeScript client ([`51623d1`](https://github.com/AlexGalhardo/vukem-finances/commit/51623d18e1d42e1aa5e500bb6574a9f075f6cb3f))

## [0.4.0] - 2026-11-22

### Added

- **telegram:** Read the bot settings ([`2dc5406`](https://github.com/AlexGalhardo/vukem-finances/commit/2dc540682b160638d4f57a58e6525710922b86cc))
- **telegram:** Add the Bot API client with escaping and token redaction ([`e1c7617`](https://github.com/AlexGalhardo/vukem-finances/commit/e1c76176811855200a5d974e7163e1bc843b23e8))
- **telegram:** Add the Telegram tables and the alert switch ([`bcc332b`](https://github.com/AlexGalhardo/vukem-finances/commit/bcc332b3e974f397ee81e4e57ae714ff8f09d931))
- **telegram:** Parse an entry from one short message ([`739936c`](https://github.com/AlexGalhardo/vukem-finances/commit/739936cd37ed91cb22c581f7d41ab16f45f56075))
- **telegram:** Add the bot texts in en, pt-BR and es ([`e1f38d8`](https://github.com/AlexGalhardo/vukem-finances/commit/e1f38d85af71a3d877f991b2abcd444eb6f23382))
- **telegram:** Handle updates, link chats and run the bot commands ([`712af30`](https://github.com/AlexGalhardo/vukem-finances/commit/712af3027feb75318aaadd185176efc4d408214f))
- **telegram:** Deliver alerts to a linked chat through alerts.Channel ([`6f59bfa`](https://github.com/AlexGalhardo/vukem-finances/commit/6f59bfaea12a64bf062fba58d346d07c3db964ee))
- **telegram:** Add telegram:set-webhook and telegram:info commands ([`abeb87d`](https://github.com/AlexGalhardo/vukem-finances/commit/abeb87d5626239c0c2925a311f2869722c1f81c3))
- **telegram:** Prune old update ids, codes, drafts and rate windows daily ([`2728fc0`](https://github.com/AlexGalhardo/vukem-finances/commit/2728fc0cb91ca0753aec2b89936a8b758bc53ab7))
- **telegram:** Add the settings page backend with connect, revoke and the alert switch ([`07e9894`](https://github.com/AlexGalhardo/vukem-finances/commit/07e9894ee5f46062022154558db22a8d48c5a5c1))
- **telegram:** Add the Telegram settings page and the i18n in three locales ([`542351f`](https://github.com/AlexGalhardo/vukem-finances/commit/542351f4f7288cefe0fe46311b19c124f193e896))
- **sync:** Add the sync_devices and sync_ops tables ([`a0da73f`](https://github.com/AlexGalhardo/vukem-finances/commit/a0da73fa63746f31bd5577ca1e4d46986a242dd0))
- **sync:** Serve the sync protocol under /api/v1/sync ([`de26bf8`](https://github.com/AlexGalhardo/vukem-finances/commit/de26bf8e9ee0385dfbb9f9fee8d62fb01bd6a203))
- **sync:** Prune the stored results of pushed operations after 30 days ([`deec748`](https://github.com/AlexGalhardo/vukem-finances/commit/deec7489b581e10df740b3f397bd2ba07f089a3a))
- **sync:** Add the Devices page ([`47aa96c`](https://github.com/AlexGalhardo/vukem-finances/commit/47aa96c9c328d4bebcdc3371b0f7fb1ca84a8ea8))
- **desktop:** Add the loopback gate of the desktop server ([`68d49ad`](https://github.com/AlexGalhardo/vukem-finances/commit/68d49adbca5b0fd110cce3ee85b1b5d21a8bde3e))
- **desktop:** Add the desktop module with paths, key and instance lock ([`9d4f1d7`](https://github.com/AlexGalhardo/vukem-finances/commit/9d4f1d7424229212cb926f050d3e2cf5200db567))
- **desktop:** Start the personal edition on a loopback port ([`0801e36`](https://github.com/AlexGalhardo/vukem-finances/commit/0801e366ec976341f4f610a14b2a3ef10ecd8763))
- **desktop:** Add the headless mode and the smoke script ([`67dd75a`](https://github.com/AlexGalhardo/vukem-finances/commit/67dd75a1981ce1bb5b39b4747a3ea09023bb5afa))
- **desktop:** Write a backup zip of the database and the key ([`ee2c618`](https://github.com/AlexGalhardo/vukem-finances/commit/ee2c61888015c753f63a7ee5451b377526882531))
- **desktop:** Add the native save and open-external routes ([`c82c231`](https://github.com/AlexGalhardo/vukem-finances/commit/c82c2315d74fd111cfd0279d2aa03f56f1d4361b))
- **desktop:** Open the app in a Wails window with a native menu ([`27eed94`](https://github.com/AlexGalhardo/vukem-finances/commit/27eed94509988f6644364ef08a6bd4c3f3fa5d1e))
- **db:** Add the tables of AI categorization and the assistant ([`2f0aa1e`](https://github.com/AlexGalhardo/vukem-finances/commit/2f0aa1eca1e36c53561a54477b5e79558219f629))
- **finance:** Remember the category of a description ([`171c44f`](https://github.com/AlexGalhardo/vukem-finances/commit/171c44fc8e55e61732a5387a2ca25dcc3de4fb11))
- **ai:** Add the batch categorizer with id and uid allowlists ([`9288062`](https://github.com/AlexGalhardo/vukem-finances/commit/9288062b9a2fb2b49d92ab00077e73e8620d9fdd))
- **ai:** Record, review and undo category suggestions ([`3cc6292`](https://github.com/AlexGalhardo/vukem-finances/commit/3cc6292018f8f6ffd30c664a7349ea699aee4ae1))
- **ai:** Enqueue and run the ai_categorize job ([`73ce383`](https://github.com/AlexGalhardo/vukem-finances/commit/73ce3830bff3ca80f8fc13edaf54887122d86df1))
- **ai:** Read a sentence into a transaction draft ([`cee0285`](https://github.com/AlexGalhardo/vukem-finances/commit/cee0285c0a4fe3e79d8320f6f2490eb658a62f3f))
- **ai:** Add the finance assistant with a validated plan and server figures ([`3caeb4c`](https://github.com/AlexGalhardo/vukem-finances/commit/3caeb4ce4f4590523fee2c3f2bbd8d2f4332b225))
- **ai:** Add the routes of suggestions, drafts and the assistant ([`c8b790d`](https://github.com/AlexGalhardo/vukem-finances/commit/c8b790de9c969fb87e0dc42c64e83bbfea1484c7))
- **ai:** Add the evaluation set and the ai:eval command ([`352f65b`](https://github.com/AlexGalhardo/vukem-finances/commit/352f65bf9a7af4bcda54a7bb9e4c4b35d4c7402b))
- **ui:** Add the suggestions screen, the assistant dialog and the sentence field ([`88bdb31`](https://github.com/AlexGalhardo/vukem-finances/commit/88bdb31c7383fa311896f67f5ccdb11c42be6c25))
- **operations:** Add the operation registry, actor, error codes and Invoke ([`e149990`](https://github.com/AlexGalhardo/vukem-finances/commit/e1499904dca876e374a00fcb39a76da9cbe70de4))
- **operations:** Replay an idempotency key once per caller ([`7747f91`](https://github.com/AlexGalhardo/vukem-finances/commit/7747f9139bad0ef937d433603cf52b375f2ec39b))
- **operations:** Audit every write and destructive call without its data ([`ae836bc`](https://github.com/AlexGalhardo/vukem-finances/commit/ae836bc83884cc7fa630e84d324bf768f63493d2))
- **api:** Add cards, statements, recurrences, goals, reports and export operations ([`f1904c5`](https://github.com/AlexGalhardo/vukem-finances/commit/f1904c54b89c34208ec98120ddd8b7f141fc1b5c))
- **operations:** Add the operations:inventory command and its committed output ([`cac92e1`](https://github.com/AlexGalhardo/vukem-finances/commit/cac92e15d28be68f9924cef6393161c812c1613d))

### Changed

- **finance:** Delete and restore synced rows inside a caller transaction ([`57f5ba1`](https://github.com/AlexGalhardo/vukem-finances/commit/57f5ba135b0e145992a894c1c3d18f9d96c13dad))
- **api:** Run accounts, categories and tags through the operation registry ([`7bcc7f6`](https://github.com/AlexGalhardo/vukem-finances/commit/7bcc7f6a6555eb317af05376ec4724dd5311d252))
- **api:** Run transactions and transfers through the operation registry ([`e5e0dab`](https://github.com/AlexGalhardo/vukem-finances/commit/e5e0dab3b6a9e65617cc0c06ea446c5847072634))
- **api:** Run budgets and summaries through the operation registry ([`1e7d5c3`](https://github.com/AlexGalhardo/vukem-finances/commit/1e7d5c32cd2fe8d1e769c7825323cfe8c5d631e3))

### Fixed

- **telegram:** Keep the undo when recording a saved entry fails ([`b50145f`](https://github.com/AlexGalhardo/vukem-finances/commit/b50145fd0f0afd25d3bf932faa24f6be9ea5442a))
- **telegram:** Move past an update that keeps failing ([`c34e8d6`](https://github.com/AlexGalhardo/vukem-finances/commit/c34e8d6cdabfde7e746ab5d82cbaaa862a68dda2))
- **telegram:** Retry for a minute and tell the chat before skipping an update ([`747d46b`](https://github.com/AlexGalhardo/vukem-finances/commit/747d46b37238e4fc3284cdf4c4c281e96141268a))
- **sync:** Derive the statement of an entry on the server ([`f28fc6b`](https://github.com/AlexGalhardo/vukem-finances/commit/f28fc6be0217095a6e037d1bc84a6810b000be38))
- **sync:** Hold restores and transfer legs to the statement rules ([`240a545`](https://github.com/AlexGalhardo/vukem-finances/commit/240a545c8816abfc63a60d2aec1c71f785c3b6c2))
- **desktop:** Keep the gate headers and load the shell script from the server ([`90f09e6`](https://github.com/AlexGalhardo/vukem-finances/commit/90f09e6bd2f5b592d33326e5ed394df37672edcc))
- **desktop:** Keep the old file when replacing a download fails ([`9f15373`](https://github.com/AlexGalhardo/vukem-finances/commit/9f153733f19cd11a992514cd5a7767ff36269242))
- **ci:** Pin NSIS to a published version ([`13f245f`](https://github.com/AlexGalhardo/vukem-finances/commit/13f245f8b3bbff48abf31f81dd1d266783e06245))
- **ai:** Normalize whitespace before redacting text sent to a provider ([`c73e731`](https://github.com/AlexGalhardo/vukem-finances/commit/c73e7310c2abcc878946dbac8dd938618186a4cc))
- **ai:** Keep account names out of the sentence to draft payload ([`3212acf`](https://github.com/AlexGalhardo/vukem-finances/commit/3212acf5a31f409a054437d3f90f61a9c0343098))
- **ai:** Send opaque labels for accounts and tags and store only sanitized questions ([`2be3fcf`](https://github.com/AlexGalhardo/vukem-finances/commit/2be3fcfcd6668110429fdcfcf1c08cb3c9de7d0c))
- **ai:** Keep the assistant, which spends the AI limit, away from viewers ([`92f1e16`](https://github.com/AlexGalhardo/vukem-finances/commit/92f1e16000bd5f8e1ad549bbc9133c7e91df4d54))
- **ai:** Let a demoted viewer read and erase their own conversations ([`3468440`](https://github.com/AlexGalhardo/vukem-finances/commit/3468440a51f3f8002e9d269c7faef32a32cf1b47))
- **operations:** Keep the idempotency claim when the answer cannot be stored ([`2cee322`](https://github.com/AlexGalhardo/vukem-finances/commit/2cee322bd89ef9d1b294737492caf884580e3a85))
- **ci:** Make the test suite and the desktop packaging reliable ([`5fd7030`](https://github.com/AlexGalhardo/vukem-finances/commit/5fd703041212eea33e74cbd20520a88692759170))

## [0.3.0] - 2026-11-22

### Added

- **cards:** Add closing, statement and due date arithmetic ([`9a690ba`](https://github.com/AlexGalhardo/vukem-finances/commit/9a690bacc5d001228c51a2c3bfcca3557025ee62))
- **cards:** Add credit_cards table and transaction statement_ref ([`edeca56`](https://github.com/AlexGalhardo/vukem-finances/commit/edeca562565d450b77257501badc0ea5face6a59))
- **cards:** Add card create, update, archive and delete ([`1000b00`](https://github.com/AlexGalhardo/vukem-finances/commit/1000b00f132f52325bca36d269423e20775d006f))
- **cards:** Tag card purchases with a statement and pay statements ([`ca339e6`](https://github.com/AlexGalhardo/vukem-finances/commit/ca339e6351b3574deb4288a9db2e5149c250b70c))
- **cards:** Add card and statement routes, handlers and pages props ([`0e62f65`](https://github.com/AlexGalhardo/vukem-finances/commit/0e62f659182dc080cb0ba090f2083271983e4328))
- **cards:** Add the cards and card statement pages with the g k action ([`48479a9`](https://github.com/AlexGalhardo/vukem-finances/commit/48479a9a10572914a3cccee6e012dc36e9b73938))
- **i18n:** Add card translations in three locales and the g k shortcut row ([`b1b84a2`](https://github.com/AlexGalhardo/vukem-finances/commit/b1b84a26d03abad0e506be3783398176ec3f431b))
- **cards:** Seed a demo card with two statements ([`07dea87`](https://github.com/AlexGalhardo/vukem-finances/commit/07dea87f64cbb859dfaba20256b81a51fee642c3))
- **api:** Add the token resource column and the vf_ prefix ([`48d1a56`](https://github.com/AlexGalhardo/vukem-finances/commit/48d1a562834d6d3f2429b5f1c5e8552599a724bb))
- **api:** Add scoped token creation, listing and revocation with the Tokens page ([`59e01dd`](https://github.com/AlexGalhardo/vukem-finances/commit/59e01ddffa44953b12b7d86ce06d487114d817ab))
- **api:** Add the bearer middleware chain, me and account endpoints ([`2c09f8a`](https://github.com/AlexGalhardo/vukem-finances/commit/2c09f8aa19865bab80977461663f9344a585d31d))
- **api:** Add category and tag endpoints with client uids and version checks ([`3aab57a`](https://github.com/AlexGalhardo/vukem-finances/commit/3aab57a2c243591eb6140154c4b961363978eda2))
- **api:** Add transaction and transfer endpoints with filters, pages, client uids and If-Match ([`568da2e`](https://github.com/AlexGalhardo/vukem-finances/commit/568da2e6c0ecc1bd40a31da4e44b745283deae02))
- **api:** Add budget and summary endpoints ([`4846d16`](https://github.com/AlexGalhardo/vukem-finances/commit/4846d1607c967b7b8c2317b5417e669b113673f0))
- **api:** Generate the OpenAPI document and the TypeScript client with freshness tests ([`3936d0e`](https://github.com/AlexGalhardo/vukem-finances/commit/3936d0e6eb08b9466a03266387e1c65a466bf30e))
- **recurrence:** Add the pure recurrence engine ([`86c8274`](https://github.com/AlexGalhardo/vukem-finances/commit/86c8274b7f45d97f9a1366898379b613fe7a7b9b))
- **recurrence:** Add the recurrences table and the transaction natural key ([`94a40a8`](https://github.com/AlexGalhardo/vukem-finances/commit/94a40a8a7fcb92d76146d23e7cf5f54182c046e7))
- **recurrence:** Create, split and delete installment purchases ([`18f1d36`](https://github.com/AlexGalhardo/vukem-finances/commit/18f1d36201443efa5396b00615a5708413e5ad01))
- **recurrence:** Generate due occurrences idempotently under the quota lock ([`e15a79d`](https://github.com/AlexGalhardo/vukem-finances/commit/e15a79dbdbde1c1aff8f58a1e1bea379c49851e3))
- **recurrence:** Run the generation job hourly ([`f999604`](https://github.com/AlexGalhardo/vukem-finances/commit/f9996043a54f209f0f7ee5bde91f89b2c92d82f0))
- **recurrence:** List what is due in the next days with virtual occurrences ([`b3fc5f6`](https://github.com/AlexGalhardo/vukem-finances/commit/b3fc5f65c5726da8cfd00808eaf2496f2f7bc66f))
- **recurrence:** Add the recurrence and installment routes and handlers ([`d1e9299`](https://github.com/AlexGalhardo/vukem-finances/commit/d1e92999c024f6f19f308bef131f8d49da2c2bf7))
- **recurrence:** Add the recurrences page, the upcoming list and the installment form ([`9d38d77`](https://github.com/AlexGalhardo/vukem-finances/commit/9d38d77e23b24883053d3a02f9f49ec2952236a6))
- **sync:** Merge a recurrence watermark by the later date ([`33af0ad`](https://github.com/AlexGalhardo/vukem-finances/commit/33af0ad97c4c34b5c2d75b9ebd4e1aa2f667c0ea))
- **db:** Add budget rollover, goals and notifications tables ([`abb0516`](https://github.com/AlexGalhardo/vukem-finances/commit/abb0516fb38d9d2bad7f4d149c9276f552db8626))
- **budgets:** Sum spending in one grouped query with rollover, parents and an overall budget ([`5641fbb`](https://github.com/AlexGalhardo/vukem-finances/commit/5641fbb044285f391ecdbbcc0a13b02df4dd077c))
- **goals:** Add goals and contributions with required monthly and projected date ([`3a8907b`](https://github.com/AlexGalhardo/vukem-finances/commit/3a8907b929797b435754fd177072ff2477d5abf0))
- **alerts:** Evaluate alerts from current data in an idempotent job ([`5bce36a`](https://github.com/AlexGalhardo/vukem-finances/commit/5bce36a9cba48ee61109b67fb2374c3d9e2ad943))
- **alerts:** Add preferences, the mail channel templates and the notification reads ([`43fc6f0`](https://github.com/AlexGalhardo/vukem-finances/commit/43fc6f0b4f9f91947651d83ee7b5911027869c33))
- **budgets:** Add goal, notification and overall budget routes with page props ([`3d0b081`](https://github.com/AlexGalhardo/vukem-finances/commit/3d0b081e26dd560b870d2d43bf5ce44400745a0a))
- **i18n:** Add budgets, goals and alerts translations and the goals shortcut ([`3d96128`](https://github.com/AlexGalhardo/vukem-finances/commit/3d9612860dc741a079dbf527fefc249c689c1f52))
- **ui:** Add the notification bell, goals page and budgets with rollover and overall ([`e032a1d`](https://github.com/AlexGalhardo/vukem-finances/commit/e032a1d6eb538f3d7c886c8fbf12bf6a168958b2))
- **export:** Format cents as a machine decimal ([`01a6d29`](https://github.com/AlexGalhardo/vukem-finances/commit/01a6d290704ace74777b3b4db3a795d0b842680b))
- **reports:** Add the report period and the category index ([`16c51e3`](https://github.com/AlexGalhardo/vukem-finances/commit/16c51e352616daed2844da296f48a3ff6d421753))
- **reports:** Add cash flow and breakdown by category, account and tag ([`0785598`](https://github.com/AlexGalhardo/vukem-finances/commit/0785598005b849fa7acfc10a68f40f3765a780b2))
- **reports:** Add the net worth series and the period comparison ([`27d6d29`](https://github.com/AlexGalhardo/vukem-finances/commit/27d6d29e497defc826537fb3a0dab9ea0421aa3f))
- **export:** Stream the transactions CSV with a formula guard ([`a3dd4e8`](https://github.com/AlexGalhardo/vukem-finances/commit/a3dd4e87b14bed1323dc3a2cc788962477968b4a))
- **export:** Add the JSON export of the workspace and its schema ([`6a4ce34`](https://github.com/AlexGalhardo/vukem-finances/commit/6a4ce349bb3e33d452a09510fd243483d6239ac5))
- **export:** Rate limit the downloads and check who may take them ([`9476959`](https://github.com/AlexGalhardo/vukem-finances/commit/9476959c0bf532551ad9d0c31adcba4ebcec8987))
- **reports:** Add the reports page, the export menu and the x and g r actions ([`b4fde1c`](https://github.com/AlexGalhardo/vukem-finances/commit/b4fde1c1cf51e40ff7611415ba7a0c63d3a61d49))

### Fixed

- **cards:** Move trashed purchases of an emptied statement with the closing day ([`a534f33`](https://github.com/AlexGalhardo/vukem-finances/commit/a534f33aa87830049525efe0ed03fcf2065c5c01))
- **api:** Answer unknown paths and methods under /api/v1 with problem+json ([`06487e0`](https://github.com/AlexGalhardo/vukem-finances/commit/06487e076a264bb5914f9f45badde9e5e7c0da2e))
- **api:** Require tags:write to create a tag through a transaction ([`f142c56`](https://github.com/AlexGalhardo/vukem-finances/commit/f142c5646a4e360389f4fdd2e881f27e86d41654))
- **billing:** Keep access on a refund of an ended period and after a won dispute ([`1c3f3d5`](https://github.com/AlexGalhardo/vukem-finances/commit/1c3f3d5417792a416aebd146a4abcd9095dad66e))
- **sync:** Read a recurrence watermark strictly and never past today ([`f7804bd`](https://github.com/AlexGalhardo/vukem-finances/commit/f7804bd954439ef366420455512a6525c3c9383a))
- **recurrence:** Bound the engine years and the walk, refuse insane dates ([`12d03a1`](https://github.com/AlexGalhardo/vukem-finances/commit/12d03a122e6039f3ef74692c03841a28f9f979ae))
- **recurrence:** Post nothing when a future rule moves into the past ([`4202a5a`](https://github.com/AlexGalhardo/vukem-finances/commit/4202a5a502f06d6f987a0c86ec6873012e3ab706))
- **billing:** Decide a refund by the event time and always finish its cancel ([`2177650`](https://github.com/AlexGalhardo/vukem-finances/commit/2177650ee047dc6b256b2568548c320b9ff9c240))
- **ui:** Name the notifications page list items so the shared bell prop is not hidden ([`0fab8bd`](https://github.com/AlexGalhardo/vukem-finances/commit/0fab8bd93cb664a6ac731ba19c0bc715f78acec0))
- **alerts:** Honor the current mail preference for pending notifications ([`06a7406`](https://github.com/AlexGalhardo/vukem-finances/commit/06a7406303f7911d3ba133ca6f228290befafe6a))
- **finance:** Detect formulas behind invisible leading characters ([`c472aa1`](https://github.com/AlexGalhardo/vukem-finances/commit/c472aa1648ea4308dfd980fc171469909fe335a2))
- **export:** Guard formulas after list separators in CSV cells ([`b2a2321`](https://github.com/AlexGalhardo/vukem-finances/commit/b2a2321010359089a179fa594f28a3ec1c77128f))
- **export:** Abort a download that fails after its first byte ([`5b5d76b`](https://github.com/AlexGalhardo/vukem-finances/commit/5b5d76b8d7eb1a09e6efc5de16022f2cfc7329f7))
- **export:** Guard formulas after line breaks in CSV cells ([`c1fa5f6`](https://github.com/AlexGalhardo/vukem-finances/commit/c1fa5f6c73f8e59607e36e2a0b859befd73b5e81))

## [0.2.0] - 2026-11-21

### Added

- **sync:** Add uid, version and tombstone columns with the change log tables ([`7d2d8c9`](https://github.com/AlexGalhardo/vukem-finances/commit/7d2d8c956180b1b929e4c31e17d63e684d10b1d3))
- **sync:** Stamp uid and version on the synced models ([`022d030`](https://github.com/AlexGalhardo/vukem-finances/commit/022d0301dfeb6bd53b23b1fcf69676ad7858e23f))
- **sync:** Add the entity registry and the change-log triggers ([`379dc1e`](https://github.com/AlexGalhardo/vukem-finances/commit/379dc1e032270e2b75d018e7cc455af1abf14c7d))
- **finance:** Soft delete synced rows and add restore ([`a5861a1`](https://github.com/AlexGalhardo/vukem-finances/commit/a5861a1e9fb7d757529f1eba57acb11523b8a2a6))
- **finance:** Cap money at 2^53 - 1 cents on every write path ([`b2269de`](https://github.com/AlexGalhardo/vukem-finances/commit/b2269deb4224c088d6b11bc036c04c3172ecc983))
- **sync:** Read the change log by cursor and take a snapshot ([`53638a1`](https://github.com/AlexGalhardo/vukem-finances/commit/53638a11a4eb21aa3d36d1c726fd9a3eac26c0c7))
- **sync:** Merge a mutation into a row by field with last-writer-wins ([`1078cf3`](https://github.com/AlexGalhardo/vukem-finances/commit/1078cf379a74fc6d96b13b864daefc89053b835a))
- **sync:** Compact tombstones and the change log with a daily job ([`340975f`](https://github.com/AlexGalhardo/vukem-finances/commit/340975f0eb8846736a3aecbce2dfeac3960d611b))
- **ai:** Add the AI tables, models and the ai.manage ability ([`d1ff415`](https://github.com/AlexGalhardo/vukem-finances/commit/d1ff41554f23cf5f651e60138cc81b7bac7f9cfb))
- **ai:** Seal user keys with workspace-bound data and re-seal after rotation ([`7866119`](https://github.com/AlexGalhardo/vukem-finances/commit/7866119507a400bd01bc2ac50cb38386cfd77031))
- **ai:** Redact personal data and key-like tokens before text leaves the server ([`3f9c50c`](https://github.com/AlexGalhardo/vukem-finances/commit/3f9c50c1a92c1a40c363136a01c231b5e8cad4f9))
- **ai:** Add the fake provider and the call pipeline with limits and usage ([`ae60f7e`](https://github.com/AlexGalhardo/vukem-finances/commit/ae60f7e03da16ee286d5ebbf255f997c8ffcd952))
- **ai:** Add the anthropic provider over the safe HTTP client ([`1b79fa2`](https://github.com/AlexGalhardo/vukem-finances/commit/1b79fa24d62511e980c438a8a11ca4eb13d1e139))
- **ai:** Add the openai and openai-compatible providers and base URL checks ([`0e877e7`](https://github.com/AlexGalhardo/vukem-finances/commit/0e877e7f36dfd365e541a7bba91185d472c65ea5))
- **ai:** Add the owner-only routes for the AI key, switches, check and usage ([`dfe4d51`](https://github.com/AlexGalhardo/vukem-finances/commit/dfe4d51acca163bb08dc6de5978e1647e0420fc7))
- **ai:** Add AI translations, the env example entry and agent notes ([`161a6ea`](https://github.com/AlexGalhardo/vukem-finances/commit/161a6eab23854f9434638d825f09b1a26b990fa0))
- **ai:** Add the AI settings page with the consent dialog and an e2e spec ([`7eb0074`](https://github.com/AlexGalhardo/vukem-finances/commit/7eb007423e1e5c739c9d30992031bbb4d1401724))
- **accounts:** Add archive, position and transfer pairing columns ([`67cad0b`](https://github.com/AlexGalhardo/vukem-finances/commit/67cad0bbef3f0531b842dd8ad69fb268271a2900))
- **accounts:** Validate kinds and add update, archive and unarchive ([`c5db6b5`](https://github.com/AlexGalhardo/vukem-finances/commit/c5db6b5e95bd09deb7782c1316d1e19e566b5fac))
- **accounts:** Derive account balances and net worth ([`8ba5e1e`](https://github.com/AlexGalhardo/vukem-finances/commit/8ba5e1ee29e30cfeae9149a69dc2c843f23eb3a5))
- **transfers:** Add two-leg transfers kept out of income, expense and quota ([`b01025a`](https://github.com/AlexGalhardo/vukem-finances/commit/b01025a99d39ccf58b0a5ebdfda065aeea176105))
- **accounts:** Add routes for account edit, archive and transfers ([`25b9f69`](https://github.com/AlexGalhardo/vukem-finances/commit/25b9f69374f5793f75410b1a61c52a44639a25f8))
- **transfers:** Serve the dialog pickers and name the other account of a transfer ([`48fd1e1`](https://github.com/AlexGalhardo/vukem-finances/commit/48fd1e15c8eed2f6bbab4c41be0f222aa708f675))
- **accounts:** Show balances, net worth and archive controls on the accounts page ([`2dc5add`](https://github.com/AlexGalhardo/vukem-finances/commit/2dc5add5df7319e28a85bec8644fbe40ece8e94e))
- **accounts:** Seed a second account and a transfer in the demo data ([`e697bab`](https://github.com/AlexGalhardo/vukem-finances/commit/e697babb4e6cf3ac09092e65df63f6392a4e9418))
- **transfers:** Add quick add dialogs for expense, income and transfer ([`a4d9ca8`](https://github.com/AlexGalhardo/vukem-finances/commit/a4d9ca81d28ab9b1a79945ad56545580b5e470aa))
- **billing:** Add revocation columns, PIX tables and billing events ([`ebed07b`](https://github.com/AlexGalhardo/vukem-finances/commit/ebed07bf1e3a8f21b3fa73fabcddcd0c8cfdc27f))
- **billing:** Derive the plan from one Entitled function ([`2150a66`](https://github.com/AlexGalhardo/vukem-finances/commit/2150a668b4817463518233dccaded83c4fcc5da3))
- **billing:** Revoke Pro on a full refund and cancel the subscription ([`e260a69`](https://github.com/AlexGalhardo/vukem-finances/commit/e260a6930c81646356652e06a38dc030416bdf89))
- **billing:** Add the AbacatePay client and PIX settings ([`35d7091`](https://github.com/AlexGalhardo/vukem-finances/commit/35d7091ee03946fd94ae298286e3c54f7841c805))
- **billing:** Add PIX checkout with idempotent activation and stacked months ([`5ff2283`](https://github.com/AlexGalhardo/vukem-finances/commit/5ff2283cde7152ac5c7155ecaa57ed96a21e9f56))
- **billing:** Add the AbacatePay webhook with secret, signature and re-check ([`73c66e1`](https://github.com/AlexGalhardo/vukem-finances/commit/73c66e19bd6d34e3af4bf11111ee764605f13444))
- **billing:** Reconcile PIX payments, revoke refunds and add the admin action ([`0d5bd76`](https://github.com/AlexGalhardo/vukem-finances/commit/0d5bd76dc9a2c4b50353d44683ca2fd94aeeace7))
- **billing:** Add the PIX start and poll routes with limits and props ([`ba33698`](https://github.com/AlexGalhardo/vukem-finances/commit/ba33698adfc442da2d21b7d637b502969928bf90))
- **billing:** Build the billing page with the PIX dialog and revocation reasons ([`efb8de0`](https://github.com/AlexGalhardo/vukem-finances/commit/efb8de0265162713fdf323939f67f4382a9d3d00))
- **billing:** Count PIX in admin revenue and add the refunded and receipt mails ([`1f879b1`](https://github.com/AlexGalhardo/vukem-finances/commit/1f879b13f354dde90d61e0163bbed10ffbbb7c4d))
- **db:** Add category tree columns, tags and the import hash ([`af06ad0`](https://github.com/AlexGalhardo/vukem-finances/commit/af06ad0c1b09f2604eb2d3c9a0ec99f4a48c4351))
- **categories:** Update, archive and delete with reassignment ([`a859368`](https://github.com/AlexGalhardo/vukem-finances/commit/a859368d79d3b74e771c0bc9f2121a3a6920f04b))
- **tags:** Create inline, limit, attach and detach ([`b3b9a9c`](https://github.com/AlexGalhardo/vukem-finances/commit/b3b9a9c10a931fe0352bee6d01fb5f72bd49eb85))
- **finance:** Search and filter transactions with pages ([`4b48c89`](https://github.com/AlexGalhardo/vukem-finances/commit/4b48c891f298d594d190b559b15553c46122c298))
- **categories:** Routes for categories, tags and the filtered list ([`225a36f`](https://github.com/AlexGalhardo/vukem-finances/commit/225a36fecdcfd9552d3a4d41466b949e007ea79d))
- **import:** Add the OFX parser with hostile-file fixtures ([`c0f2e50`](https://github.com/AlexGalhardo/vukem-finances/commit/c0f2e50abff16101177d2a5a134b7ee00ffb52e6))
- **import:** Detect duplicates, preview totals and store a report ([`eaca074`](https://github.com/AlexGalhardo/vukem-finances/commit/eaca074797a0ee8d260f8461f096fb2f4158fb94))
- **i18n:** Add categories, search and import strings in three locales ([`8876f28`](https://github.com/AlexGalhardo/vukem-finances/commit/8876f28b996c465537e987614cf748cec5c63966))
- **ui:** Categories tree, tags, filters, search action and import flow ([`92d1f61`](https://github.com/AlexGalhardo/vukem-finances/commit/92d1f61ea644944c33457f2123644e669650f94d))

### Changed

- **finance:** Split models, routes and handlers by domain ([`fa0287c`](https://github.com/AlexGalhardo/vukem-finances/commit/fa0287cc19bae72f6d834164e1fc921e737f96e9))
- **import:** Read statements through StatementParser ([`ed0f675`](https://github.com/AlexGalhardo/vukem-finances/commit/ed0f675d95d904aebaf3c496bceeaf8033ed7439))

### Fixed

- **sync:** Read ISO timestamps when backfilling uids and stamp the log the same way ([`81d02b6`](https://github.com/AlexGalhardo/vukem-finances/commit/81d02b6cb7ced8fcd49a114534109bc648aa567f))
- **sync:** Keep the cursor at the compacted seq so a snapshot cannot loop ([`d7c6932`](https://github.com/AlexGalhardo/vukem-finances/commit/d7c693293effc37533db3420cd897ca52ee58e19))
- **ai:** Use the current Claude model and treat refused or cut-off answers as errors ([`01483db`](https://github.com/AlexGalhardo/vukem-finances/commit/01483dbde110504ed9f4222edbc2873cc3ccc5c1))
- **ai:** Close the usage reservation when the client disconnects ([`06ec9c1`](https://github.com/AlexGalhardo/vukem-finances/commit/06ec9c13dece74cb0d7934b9b50f3cc40fa569d4))
- **ai:** Keep billed and unknown-outcome calls counted against the monthly limit ([`6fb5098`](https://github.com/AlexGalhardo/vukem-finances/commit/6fb509858604803c5cf984edfe59acd73905b554))
- **accounts:** Refuse new entries on archived accounts ([`d24c42f`](https://github.com/AlexGalhardo/vukem-finances/commit/d24c42f5a391878dc1c2653d846b7ef7464881d6))
- **docker:** Cross-compile the image instead of emulating arm64 ([`9bedfc9`](https://github.com/AlexGalhardo/vukem-finances/commit/9bedfc9f1e58bbf22855b2c3e6bb8076fd42096a))
- **billing:** Read the plan inside the transaction that checks the quota ([`ab23e52`](https://github.com/AlexGalhardo/vukem-finances/commit/ab23e52a56e35e23d9324bfbea2e33e508594047))
- **billing:** Keep the current PIX period when an ended one is refunded ([`01652b4`](https://github.com/AlexGalhardo/vukem-finances/commit/01652b4c94a9a40ef5da647ba1d53ac1d1f2326f))
- **billing:** Give back only the months a PIX dispute withheld ([`bb98be7`](https://github.com/AlexGalhardo/vukem-finances/commit/bb98be7ae77215b3f3ca402cfbb31ed1df3c1ade))
- **billing:** Restore a won PIX dispute on its original term ([`bc477b9`](https://github.com/AlexGalhardo/vukem-finances/commit/bc477b9fae9af34d998ad8abaaa021ee8298b399))
- **billing:** Recompute PIX months from paid payments on a won dispute ([`2a15344`](https://github.com/AlexGalhardo/vukem-finances/commit/2a1534417dcbf11fb31c3385210016c3bc5c727a))
- **billing:** Keep PIX paid months exact across a lapsed period ([`b0e7e16`](https://github.com/AlexGalhardo/vukem-finances/commit/b0e7e167356a97155098e83f7eee616072f08a19))
- **finance:** Refuse amounts with malformed separators or a second number ([`11703d8`](https://github.com/AlexGalhardo/vukem-finances/commit/11703d861418868dcdbcc5f3d0fe8dc52bd3e2fa))
- **finance:** Keep restored transactions off deleted categories and tags ([`72b3a26`](https://github.com/AlexGalhardo/vukem-finances/commit/72b3a26acf93408aa737e6807a8fe5e49b4881a8))

## [0.1.0] - 2026-11-21

### Added

- **edition:** Add APP_EDITION loading ([`ddc9adc`](https://github.com/AlexGalhardo/vukem-finances/commit/ddc9adce94af686cfd707e0aaa22e3bc76ed6980))
- **edition:** Stop boot on an invalid APP_EDITION ([`71f4817`](https://github.com/AlexGalhardo/vukem-finances/commit/71f4817a1bdf596daab4d75a57c76b1d4ddc9b23))
- **edition:** Lift every plan cap in the personal edition ([`2f1f7b4`](https://github.com/AlexGalhardo/vukem-finances/commit/2f1f7b4dad59356a3906c3a49f25c56c872fc28c))
- **edition:** Leave billing and revenue out of the personal edition ([`cb7251e`](https://github.com/AlexGalhardo/vukem-finances/commit/cb7251ecda77d455b3b95c7620f7f3cf86160048))
- **edition:** Close public sign-up and add the setup flow in personal ([`f29227d`](https://github.com/AlexGalhardo/vukem-finances/commit/f29227d8691896af15f88ba1f9b7b90cfd2289e2))
- **edition:** Register through an invite link in personal ([`8a2d71e`](https://github.com/AlexGalhardo/vukem-finances/commit/8a2d71ed52612849a484b93970a33c8d87f0f5b4))
- **edition:** Make social login find-only in personal ([`e5b6853`](https://github.com/AlexGalhardo/vukem-finances/commit/e5b6853b392bd3528fc77ee83cb66ed7e7cd6828))
- **edition:** Keep a single implicit workspace in personal ([`1fdac5a`](https://github.com/AlexGalhardo/vukem-finances/commit/1fdac5a6eb1e25d42e3248f439d57b736886f20e))
- **edition:** Hide the landing, switcher and billing entry in personal ([`eb37b0a`](https://github.com/AlexGalhardo/vukem-finances/commit/eb37b0ab0de6bc8b7a81c3d29bb07c2b918caa56))
- **docker:** Stamp the release version in the image and show it in about ([`e99918b`](https://github.com/AlexGalhardo/vukem-finances/commit/e99918bbd5fa09fc594666f6aedac85f51721a9a))
- **docker:** Add the VPS stack with Caddy and automatic HTTPS ([`d98b09c`](https://github.com/AlexGalhardo/vukem-finances/commit/d98b09cd712ee8d7d006190e7a26010d828177a2))
- **cli:** Add user:reset-password for a locked-out owner ([`db10732`](https://github.com/AlexGalhardo/vukem-finances/commit/db10732f8fa00f4f6d5539c8af4284ac831b1e9f))
- **docker:** Let compose.prod.yaml run a published image through APP_IMAGE ([`15fd162`](https://github.com/AlexGalhardo/vukem-finances/commit/15fd162f9e8921791a1568ecefbec706bb44d916))
- **workspace:** Add the read-only viewer role ([`9f3be4c`](https://github.com/AlexGalhardo/vukem-finances/commit/9f3be4c0d4933b66032265369308be60dcdcab11))
- **workspace:** Change roles, transfer ownership and leave ([`1287bbf`](https://github.com/AlexGalhardo/vukem-finances/commit/1287bbfbac8fddec97a4917191daafb190aead53))
- **workspace:** Add a validated timezone to each workspace ([`43d9d37`](https://github.com/AlexGalhardo/vukem-finances/commit/43d9d37ab7ff39772baccc510e6859135cab2812))
- **finance:** Send the monthly summary in each workspace's local time ([`b4b33de`](https://github.com/AlexGalhardo/vukem-finances/commit/b4b33decb01f12295276c1168982659f9b4a4d44))
- **workspace:** Rate limit the workspace routes ([`da4b8c5`](https://github.com/AlexGalhardo/vukem-finances/commit/da4b8c5a21c7465150c885536303bd11c05d1509))
- **workspace:** Record sensitive workspace actions in an audit log ([`a65968d`](https://github.com/AlexGalhardo/vukem-finances/commit/a65968d2f4531d832b0e38f67414d2ea9664bfc5))
- **ui:** Add design tokens with light and dark themes and a contrast test ([`ffff394`](https://github.com/AlexGalhardo/vukem-finances/commit/ffff39463d14a2bfe2fd37337f27ad4d5c50dbb3))
- **ui:** Add the shortcut registry, key matcher, typing guard and reserved list ([`5016ea7`](https://github.com/AlexGalhardo/vukem-finances/commit/5016ea784bc512aeb725df54b6e12c0c93834772))
- **ui:** Add PageHeader, MoneyText, DataTable, Tabs, Pagination, Toast and Combobox ([`79e1a69`](https://github.com/AlexGalhardo/vukem-finances/commit/79e1a69782c271ef656800efda44e77c87559942))
- **ui:** Add the command palette, the shortcut help and the navigation actions ([`c3fa86b`](https://github.com/AlexGalhardo/vukem-finances/commit/c3fa86b67e94a06f66cbc86433d55f7afec6184e))
- **ui:** Replace AppLayout with the Shell, a skip link and focus on navigation ([`7174046`](https://github.com/AlexGalhardo/vukem-finances/commit/7174046f61d6f00c3b633796e6027924088a0503))

### Changed

- **i18n:** Split the dictionaries per locale and domain with a parity test ([`453203a`](https://github.com/AlexGalhardo/vukem-finances/commit/453203a4e494ca2f8f060038d161f861448ff057))
- **ui:** Use PageHeader on the signed-in pages and format the code with Biome ([`c9aa86f`](https://github.com/AlexGalhardo/vukem-finances/commit/c9aa86f5b8e2997ed718e6122032a10873a838ac))

### Fixed

- **accounts:** Fail closed on email verification in saas ([`827c6f6`](https://github.com/AlexGalhardo/vukem-finances/commit/827c6f6e6662b75dbc567360fbba570a85007a7e))
- **edition:** Answer 404 on /admin/revenue in the personal edition ([`a3903a5`](https://github.com/AlexGalhardo/vukem-finances/commit/a3903a56f644fad270f16c080e6bee94b7551a11))
- **docker:** Create the backups and restore directories in the storage volume ([`a439248`](https://github.com/AlexGalhardo/vukem-finances/commit/a4392486863f9d714ff857fede1c2a1b24d053bf))
- **finance:** Read today and the month in the workspace timezone ([`4773894`](https://github.com/AlexGalhardo/vukem-finances/commit/4773894787f7e203b7fe90c832aeb460894d5d5c))
- **finance:** Clamp month arithmetic to the last day of the month ([`83364c6`](https://github.com/AlexGalhardo/vukem-finances/commit/83364c6e7c4d42da3b2f513f2d339a58e9b46c0c))
- **security:** Limit failed two-factor codes per account ([`da9fbb1`](https://github.com/AlexGalhardo/vukem-finances/commit/da9fbb1434701abbc55154d48d5b25f6dc4b7025))
- **security:** Hold the two-factor budget under concurrent requests ([`9b7d9a9`](https://github.com/AlexGalhardo/vukem-finances/commit/9b7d9a906d175d67385c6e6223018f48a03bc25a))
- **finance:** Keep sending monthly summaries after one workspace fails ([`09e4317`](https://github.com/AlexGalhardo/vukem-finances/commit/09e4317baa6445a9e55eb176e641e0fae79cd5be))
- **ui:** Return focus to the opener, focus the heading and mark only the active nav item ([`be321fe`](https://github.com/AlexGalhardo/vukem-finances/commit/be321fe5003d280287a929aafea6b4744ab8067a))
- **ci:** Pass the config to the secret scan ([`af02846`](https://github.com/AlexGalhardo/vukem-finances/commit/af02846aafd8f11b5d9e4ba0ccd0380acae7ef43))
- **release:** Keep the generated changelog free of extra blank lines ([`2925ac7`](https://github.com/AlexGalhardo/vukem-finances/commit/2925ac7a484dbb283c9a9970c5d1c3fcaa9ead0e))

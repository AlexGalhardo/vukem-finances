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

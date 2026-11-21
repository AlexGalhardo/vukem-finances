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

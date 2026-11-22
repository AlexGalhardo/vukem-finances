# Desktop app

vukem-finances ships as a desktop application for Windows, macOS and Linux: the
personal edition (one implicit workspace, no billing, no public sign-up) in a
native window. It works offline (see [offline.md](offline.md) for every network path), keeps its data in your own user folder and
needs no Docker, no terminal and no browser tab.

Builds are **unsigned** (see [Unsigned builds](#unsigned-builds-and-os-warnings)).

## How it works

One executable starts the same Go server as the web app on `127.0.0.1` with a
free port, behind a local gate, and shows it in a Wails v3 window. Closing the
window quits the app and stops the server.

The gate (`app/desktopgate`) protects the port from other programs on the same
machine:

- A request must come from a loopback address, carry a loopback `Host` (this
  defeats DNS rebinding), carry no proxy headers, and be a same-origin fetch.
- It must also carry the `vf_desktop` cookie (HttpOnly, SameSite=Strict). The
  cookie is minted by a one-time exchange of a 256-bit launch token that the
  window opens on start. The token works once; every other request, `/up`
  included, answers 403.
- Responses carry `Referrer-Policy: no-referrer` and `frame-ancestors 'none'`.

First run shows `/setup`; later runs show the login form. The setup code of the
server edition is not asked: the gate already proves the request comes from the
window. A session lasts 30 days.

Only one instance runs per data folder: a second launch focuses the first
window (and an OS file lock on `<data>/lock` stops any other process from
writing the same database).

## Where the data lives

The data folder is `vukem-finances` inside the user config directory:

| OS | Folder |
| --- | --- |
| Windows | `%AppData%\vukem-finances` |
| macOS | `~/Library/Application Support/vukem-finances` |
| Linux | `~/.config/vukem-finances` |

| Path | What it holds |
| --- | --- |
| `database.sqlite` | the SQLite database (WAL mode) |
| `app.key` | the application key (`APP_KEY`) |
| `logs/app.log` | JSON logs, rotated at 5 MiB, 3 files kept |
| `backups/` | pre-migration copies (see below) |
| `webview/` | the browser profile of the window (Windows) |
| `lock` | the single-instance lock |

The desktop app reads **no** `.env` file and no environment variable of your
shell: its configuration is fixed (`APP_ENV=production`, `APP_EDITION=personal`,
`MAIL_MAILER=log`, JSON logs). Uninstalling does not delete this folder.

## The application key

`app.key` is created on the first run from 32 random bytes (mode `0600`; on
Windows an access list with only your user). It is never regenerated: if the
file exists but is empty or damaged, the app refuses to start instead of
replacing it. **Without the key, encrypted fields (AI provider keys)
cannot be read.** Back it up together with the database.

## Backups

- **File, Export backup...** writes a zip with a consistent copy of the
  database and `app.key`, through the native save dialog. The zip holds your
  financial data and your key: keep it private.
- Before applying a migration to a database that already has data (an
  upgrade), the app copies it to
  `backups/pre-migrate-<version>-<time>.sqlite`. The newest 3 copies stay.
- **File, Open data folder** opens the folder in your file manager.

To restore, quit the app, put `database.sqlite` and `app.key` from the zip in
the data folder (delete `database.sqlite-wal` and `-shm` if present) and start
it again.

## Downloads and exports

The export links of the web app (CSV, JSON) have the `download` attribute. The
window sends them through the native save dialog: a script that the shell adds
to the pages (`/desktop/shell.js`) fetches the file and posts the bytes to
`POST /desktop/save`, and Go asks for the destination and writes it. External
links (any other origin) open in the system browser, never in the window; only
`http`, `https` and `mailto` addresses are opened.

What was observed: on Windows (WebView2) the route is used for every download,
so the native dialog appears. WKWebView (macOS) and WebKitGTK (Linux) may not
save an attachment by themselves, which is why the same route is used on every
platform. Those two were not run on a real machine: they are covered by CI
builds and the headless smoke test only.

## Unsigned builds and OS warnings

The installers are not code-signed or notarized (that needs certificates the
author has not bought). Expect warnings:

- **Windows**: SmartScreen shows "Windows protected your PC". Choose "More
  info", then "Run anyway". The installer is per-user and needs no
  administrator rights.
- **macOS**: Gatekeeper says the app cannot be opened. Open System Settings,
  Privacy & Security, and choose "Open Anyway" for vukem-finances (or
  right-click the app and choose Open). The app is only ad-hoc signed.
- **Linux**: install the `.deb` (`sudo apt install ./vukem-finances_*.deb`) or
  make the AppImage executable and run it. They need GTK 4 and WebKitGTK 6.0
  (Ubuntu 24.04+, Debian 13+).

## Headless mode

`vukem-finances-desktop --headless --print-url [--data-dir <folder>]` starts
only the server and the gate, with no window, and prints the entry URL (with the
one-time token) on stdout. CI uses it: `bash desktop/scripts/smoke.sh` follows
the entry URL, checks that a request without the cookie gets 403 (a foreign
`Host` too), that the token works once and that `/setup` loads, then stops the
process and checks that the port no longer accepts connections.

## Build

Requirements: Go (see `go.mod`), Bun, the `wails3` tool at the same version as
the module, `task`, and per platform:

- Windows: [NSIS](https://nsis.sourceforge.io) (`makensis` in `PATH`).
- Linux: `libgtk-4-dev`, `libwebkitgtk-6.0-dev`, `build-essential`.
- macOS: Xcode command line tools.

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
cd desktop
wails3 package VERSION=0.1.0
```

`desktop/` is its own Go module (`replace` to the parent), so the root module
and the Docker image do not change. `wails3 package` builds the web assets,
links `config.Version` and writes to `desktop/bin/`:

| OS | Files |
| --- | --- |
| Windows | `vukem-finances-<version>-windows-amd64-setup.exe` (NSIS, per user), `...-portable.exe` |
| macOS | `vukem-finances-<version>-macos-<arch>.dmg` (drag to Applications) and `...-macos-<arch>.zip`; `<arch>` is the arch of the runner (`macos-latest` is Apple silicon, so arm64 only) |
| Linux | `vukem-finances_<version>_<arch>.deb`, an AppImage |

`.github/workflows/desktop.yml` builds the three platforms on a published
release (and on manual dispatch), runs the smoke test and uploads the files to
the release.

## Microsoft Store (MSIX)

The Windows app can also be packaged as an MSIX for the Microsoft Store. The
Store signs the package it publishes, so users never see the SmartScreen
warning of the other installers. The MSIX is **not** attached to the GitHub
release: only the Store gets it.

**The MSIX steps of CI are off.** They run only when the repository variable
`MSIX_ENABLED` is `true`, which also gates the Store job. Decide where the data
folder lives before turning them on, because Windows can delete the data of a
packaged app on uninstall.

### Build one locally

Needs the Windows SDK (`makeappx.exe`). Pack the exe that `wails3 package`
leaves in `desktop/bin/`:

```sh
cd desktop
wails3 task windows:msix VERSION=0.5.0
```

The output is `bin/vukem-finances-<version>-windows-amd64.msix`, unsigned, with
a **fake identity** (`VukemFinances.CI`), so it can never be submitted by
mistake. For the real identity pass the three values of Partner Center:
`MSIX_IDENTITY_NAME`, `MSIX_PUBLISHER` (`CN=...`) and
`MSIX_PUBLISHER_DISPLAY_NAME`. The package version is `X.Y.Z.0` (the Store
reserves the last part); a pre-release suffix is dropped.

The manifest template and the logos are in `desktop/build/windows/msix/`. The
logos come from `desktop/build/appicon.png`; run `assets.ps1` there to
regenerate them. The manifest declares `rescap:Capability runFullTrust` (a
Win32 app), which Partner Center asks you to justify (see the certification
notes).

### What CI does

Every run of the Desktop workflow, on Windows and without any secret:

1. packs the MSIX with the fake identity;
2. signs it with a self-signed certificate created in the job (the private key
   stays in the runner's certificate store, only the public part is trusted,
   nothing is written to the artifact) and removes it at the end;
3. installs it, runs `desktop/scripts/smoke.sh` on the installed executable and
   uninstalls it;
4. runs the Windows App Certification Kit (`appcert.exe`) when the runner has it.
   This step does not fail the run yet, because it was never run on a hosted
   runner when it was written. Read its result in the first run and then
   remove `continue-on-error`;
5. uploads the package and the kit report as the artifact `msix-ci-test-signed`.

That package is test-signed: it installs only on a machine that trusts the
throwaway certificate, so it is never published. The smoke test runs the
installed executable headless, without package identity; it proves the package
installs and carries a working exe, not that the window opens.

### First submission (manual)

The Store API only updates a product that is already live, and only free ones.
Do the first submission by hand:

1. Register in [Partner Center](https://partner.microsoft.com/dashboard) as a
   Windows app developer. Check the account type first: Store policy 10.8.3
   asks for a company account when the main function of the product needs
   financial information, and that may apply to vukem-finances.
2. Reserve the app name. Under Product management, Product identity, copy
   *Package/Identity/Name*, *Package/Identity/Publisher* and
   *Package/Properties/PublisherDisplayName*.
3. Build the package with those values (see above) from the release exe and
   upload it in a new submission. Fill in the age rating questionnaire, the
   privacy policy URL, the listing and the certification notes below. A
   submission with the first package is what turns the product into one the
   API can update.
4. Wait for the certification and publication.

### Automatic submission of the next releases

The `store` job of `.github/workflows/desktop.yml` runs on a published release
that is not a pre-release, and only when the repository variables `MSIX_ENABLED`
and `MSSTORE_ENABLED` are `true`. Otherwise it is skipped. It packs the exe that the
Windows job built (not signed), then uses the
[Microsoft Store Developer CLI](https://learn.microsoft.com/en-us/windows/apps/publish/msstore-dev-cli/github-actions)
(`msstore`, pinned in the workflow) to upload and commit the submission and to
poll it. The job fails when the submission fails or is rejected, and its
timeout is 5 hours: a longer certification does not cancel the submission, it
only stops the wait.

Register an app in Microsoft Entra ID, add it in Partner Center (Account
settings, User management, Microsoft Entra applications) with the **Manager**
role, then create in the repository:

| Kind | Name | Value |
| --- | --- | --- |
| Secret | `AZURE_AD_TENANT_ID` | Tenant ID of the Entra directory |
| Secret | `AZURE_AD_APPLICATION_CLIENT_ID` | Application (client) ID of the registration |
| Secret | `AZURE_AD_APPLICATION_SECRET` | A client secret of the registration |
| Secret | `SELLER_ID` | Seller ID of the Partner Center account |
| Variable | `MSSTORE_PRODUCT_ID` | Store product ID (the 12 character ID of the product) |
| Variable | `MSSTORE_IDENTITY_NAME` | Package/Identity/Name |
| Variable | `MSSTORE_PUBLISHER` | Package/Identity/Publisher (`CN=...`) |
| Variable | `MSSTORE_PUBLISHER_DISPLAY_NAME` | Package/Properties/PublisherDisplayName |
| Variable | `MSIX_ENABLED` | `true` to turn on the MSIX steps of CI |
| Variable | `MSSTORE_ENABLED` | `true`, with `MSIX_ENABLED`, to turn the job on |

The credentials reach the job through `env`, never through the script text.
Each release must have a higher `X.Y.Z` than the last one submitted.

### Certification notes

Paste this in the "Notes for certification" of the submission:

- The app is a Win32 desktop app (`runFullTrust`). It needs full trust because
  it keeps its data in the user's profile, opens native save and folder
  dialogs and starts its own local web server.
- It listens only on `127.0.0.1`, on a random free port, never on another
  interface, and opens no port to the network. The window is a WebView2 that
  loads that local address. The server answers nothing without a cookie that
  the window obtains once, with a single-use 256-bit launch token, so other
  programs of the machine cannot use it. It needs no internet connection and
  no account.
- No sign-in is needed to test it: the first launch shows the setup form that
  creates the local user.
- Data lives in `%AppData%\vukem-finances`; the app is offline-first and
  sends no telemetry.

Policy 10.8.3 (financial information) and the loopback server were not
checked by Microsoft when this was written: expect questions in the first
review and answer them with the paragraphs above.

### Open points

- **Data and uninstall.** A packaged full-trust app that writes under
  `%AppData%` has those writes redirected into the package's own folder, which
  Windows deletes on uninstall. That would delete the database and `app.key`
  (the unpackaged installers never do). It was not run on a real machine.
  Avoiding it needs the restricted capability `unvirtualizedResources` and a
  justification to Microsoft, or moving the data folder: a decision for the
  maintainer before the first submission.
- **WebView2.** The runtime is part of Windows 11 and of current Windows 10, but
  the MSIX cannot carry the bootstrapper that the NSIS installer bundles.
- **Version.** Check in the first submission that Partner Center accepts a
  `0.x.y.0` package version.
- **Wails.** `wails3 tool msix` (v3.1.0) exists and also uses MakeAppx, but it
  generates its own manifest and placeholder logos; this repository packs with
  its own script to own the manifest and the identity inputs.

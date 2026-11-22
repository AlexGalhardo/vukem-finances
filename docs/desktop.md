# Desktop app

vukem-finances ships as a desktop application for Windows, macOS and Linux: the
personal edition (one implicit workspace, no billing, no public sign-up) in a
native window. It works offline, keeps its data in your own user folder and
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
replacing it. **Without the key, encrypted fields (AI provider keys, sessions)
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
| macOS | `vukem-finances-<version>-macos-<arch>.zip` |
| Linux | `vukem-finances_<version>_<arch>.deb`, an AppImage |

`.github/workflows/desktop.yml` builds the three platforms on a published
release (and on manual dispatch), runs the smoke test and uploads the files to
the release.

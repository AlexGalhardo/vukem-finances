---
title: "Desktop app"
description: "The desktop app runs the personal edition in a native window on Windows, macOS and Linux, with its data in your user folder and no internet needed."
---

vukem-finances ships as a desktop application for Windows, macOS and Linux. It is the personal edition in a native window: one implicit workspace, no billing and no public sign-up. It needs no Docker, no terminal and no browser tab, and it works offline.

## Install

Download the installer for your system from [GitHub Releases](https://github.com/AlexGalhardo/vukem-finances/releases):

| System | Files |
| --- | --- |
| Windows | a per-user setup `.exe` and a portable `.exe` |
| macOS | a `.dmg` (drag to Applications) and a `.zip` |
| Linux | a `.deb` and an AppImage, which need GTK 4 and WebKitGTK 6.0 (Ubuntu 24.04 or newer, Debian 13 or newer) |

The builds are unsigned, so the system warns you:

- **Windows.** SmartScreen shows "Windows protected your PC". Choose "More info", then "Run anyway". No administrator rights are needed.
- **macOS.** Gatekeeper says the app cannot be opened. Open System Settings, Privacy and Security, and choose "Open Anyway", or right-click the app and choose Open.
- **Linux.** Install the `.deb` with `sudo apt install ./vukem-finances_*.deb`, or make the AppImage executable and run it.

## How it works

One executable starts the same Go server as the web app on `127.0.0.1` with a free port, behind a local gate, and shows it in a window. Closing the window quits the app and stops the server. The gate accepts only requests that come from the window itself, so other programs on your machine cannot talk to the port.

The first run shows `/setup`; later runs show the sign-in form. The setup code of the server edition is not asked, because the gate already proves the request comes from the window.

## Where the data lives

The data folder is `vukem-finances` inside your user configuration directory:

| System | Folder |
| --- | --- |
| Windows | `%AppData%\vukem-finances` |
| macOS | `~/Library/Application Support/vukem-finances` |
| Linux | `~/.config/vukem-finances` |

It holds the SQLite database and `app.key`, the application key created on the first run. Without that key the encrypted fields cannot be read, so keep it together with the database. Uninstalling does not delete the folder.

## Backups

- **File, Export backup** writes a zip with a consistent copy of the database and the key, through the native save dialog. The zip holds your financial data and your key: keep it private.
- Before applying a migration to a database that already has data, the app copies it to `backups/pre-migrate-<version>-<time>.sqlite` and keeps the newest three copies.
- **File, Open data folder** opens the folder in your file manager.

To restore, quit the app, put `database.sqlite` and `app.key` from the zip in the data folder (delete `database.sqlite-wal` and `database.sqlite-shm` if present) and start it again.

## Offline and updates

The desktop app makes no connection unless you turn an integration on, and it has no updater: a new version is a new installer. See [offline use](/docs/run/offline/) and [upgrading](/docs/run/upgrading/).

More detail, including the build and the headless mode, is in `docs/desktop.md` in the repository.

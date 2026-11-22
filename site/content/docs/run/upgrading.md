---
title: "Upgrading"
description: "Update a Docker installation to a new release, or the desktop app to a new installer, and what to do when you need to go back."
---

Releases are tagged `vX.Y.Z`. The image is published as `ghcr.io/alexgalhardo/vukem-finances:X.Y.Z` and on Docker Hub as `alexgalhardo/vukem-finances`. Take a [backup](/docs/run/backup-restore/) first, then choose one way.

## Docker on a VPS

From the published image:

```sh
export APP_IMAGE=ghcr.io/alexgalhardo/vukem-finances:0.2.0
docker compose -f compose.prod.yaml pull app
docker compose -f compose.prod.yaml up -d --no-build --wait
```

Or build the tag yourself:

```sh
git fetch --tags && git checkout v0.2.0
VERSION=0.2.0 docker compose -f compose.prod.yaml up -d --build --wait
```

The app applies new migrations when it starts (`/app/app migrate` runs before the server). To see the running version:

```sh
docker compose -f compose.prod.yaml exec app /app/app about
```

## Docker on localhost

Fetch the new code and rebuild:

```sh
git pull
docker compose up -d --build --wait
```

## Desktop app

The desktop app has no updater. Install the new installer over the old one. Before it applies a migration to a database that already has data, the app copies the database to `backups/pre-migrate-<version>-<time>.sqlite` in the data folder and keeps the newest three.

## Going back

Migrations do not run backwards. To go back, restore the backup you took before the upgrade and start the older version; see [backup and restore](/docs/run/backup-restore/).

## What changed

Every release has a page in the [changelog](/docs/changelog/).

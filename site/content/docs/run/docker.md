---
title: "Docker on localhost"
description: "Run the personal edition on your own computer with Docker Compose: one image, one SQLite file on a volume, a local mail inbox and no internet needed."
---

`compose.yaml` runs the app on your own computer. It is one image and one SQLite file on a volume, with no database server, plus a mail catcher so every email the app sends lands in a local inbox.

## What the stack contains

| Service | Address | What it is |
| --- | --- | --- |
| app | `127.0.0.1:3000` | the application |
| mailpit | `127.0.0.1:8025` | the inbox for every email the app sends |

Both ports are bound to the loopback address only, so nothing is reachable from other machines.

## Start it

```sh
cp .env.example .env
docker compose up -d --build --wait
```

Set `APP_KEY` in `.env` first (`openssl rand -base64 32` makes a good one). The edition is `personal` unless you choose another one:

```sh
APP_EDITION=saas docker compose up -d --build --wait
```

The `saas` edition also needs the `STRIPE_*` settings; see [the editions](/docs/start/editions/).

## Create the owner

The first owner of a personal installation is created at `/setup`. It asks for `SETUP_TOKEN` from `.env` or, when that is empty, for the one-time code the server prints at boot:

```sh
docker compose logs app | grep "setup code"
```

## Stop it

```sh
docker compose down
```

The data stays in the `storage` volume. `docker compose down -v` also deletes the volume, which deletes your data.

## Offline

The image has everything it needs: the page, scripts, styles and fonts come from the binary, with no CDN. With the network cut the stack still starts and every core feature works; [offline use](/docs/run/offline/) lists what needs the internet and why.

## Next steps

- Make copies of your data: [backup and restore](/docs/run/backup-restore/).
- Move to a server with HTTPS: [a VPS with Caddy](/docs/run/vps/).
- The full deployment guide is `deploy/README.md` in the repository.

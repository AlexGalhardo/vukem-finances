---
title: "Backup and restore"
description: "Make a consistent snapshot of the SQLite database while the app runs, copy it off the server and restore it safely."
---

The data is one SQLite file, and the app has a command that copies it safely while it runs. Never copy the live file by hand.

## Make a backup

`db:backup` writes a consistent snapshot with `VACUUM INTO` while the app keeps running, to a path that does not exist yet:

```sh
docker compose -f compose.prod.yaml exec app /app/app db:backup -to /app/storage/backups/$(date +%F).sqlite
```

For the local stack leave out `-f compose.prod.yaml`. The snapshot is a normal SQLite file. It lives in the same volume as the database, so also copy it off the server:

```sh
docker compose -f compose.prod.yaml cp app:/app/storage/backups/2026-11-21.sqlite ./
```

A daily cron entry on the host can do both. Keep the copies somewhere that is not this server. Old snapshots are not removed for you.

On Windows with Git Bash, put `MSYS_NO_PATHCONV=1` before the command: Git Bash otherwise turns `/app/...` into a Windows path.

## Restore

Restore into a directory other than the snapshot's, and never over the live file while the app runs:

```sh
C="docker compose -f compose.prod.yaml"
$C stop app
$C run --rm --no-deps --entrypoint /app/app app \
  db:backup -restore -from /app/storage/backups/2026-11-21.sqlite -to /app/storage/restore/database.sqlite
$C run --rm --no-deps --entrypoint rm app -f /app/storage/database.sqlite-wal /app/storage/database.sqlite-shm
$C run --rm --no-deps --entrypoint mv app /app/storage/restore/database.sqlite /app/storage/database.sqlite
$C up -d --wait app
```

The third command removes the leftover write-ahead files of the old database so SQLite does not replay them over the restored file. To restore a snapshot that is not in the volume yet, copy it in first with `docker compose -f compose.prod.yaml cp ./snapshot.sqlite app:/app/storage/backups/`. The image has no `sqlite3` binary on purpose. After a restore, open the app and check your data.

## Do not forget the key

`APP_KEY` encrypts sensitive fields, such as an AI provider key. A restored database needs the same key. Store it with your backups, in a safe place.

## Other copies of your data

- The desktop app has File, Export backup; see [the desktop guide](/docs/run/desktop/).
- The mobile app shares a JSON file of its database; see [the mobile app](/docs/run/mobile/).
- You can always take your data out as CSV or JSON; see [reports and exports](/docs/use/insights-exports/).

# Deploying vukem-finances with Docker

One image, one SQLite file on a volume, no database server. Two stacks:

| File | For | Ports |
| --- | --- | --- |
| `compose.yaml` | your own computer: the app and a mail catcher | `127.0.0.1:3000` (app), `127.0.0.1:8025` (mail inbox) |
| `compose.prod.yaml` | a VPS: the app behind Caddy with automatic HTTPS | 80 and 443 (Caddy only; the app has no published port) |

Both keep the SQLite database in the `storage` volume (`/app/storage/database.sqlite`).

## On your computer

```sh
cp .env.example .env          # then set APP_KEY (openssl rand -base64 32)
docker compose up -d --build --wait
```

Open <http://localhost:3000>. The edition is `personal` unless you choose another:
`APP_EDITION=saas docker compose up -d --build --wait` (the `saas` edition needs the
`STRIPE_*` settings in `.env`).

The first owner of a `personal` install is created at `/setup`. It asks for `SETUP_TOKEN`
from `.env` or, when that is empty, for the one-time code the server prints at boot:

```sh
docker compose logs app | grep "setup code"
```

Every email the app sends is in the mail inbox at <http://localhost:8025>. Stop with
`docker compose down` (the data stays); `docker compose down -v` also deletes the volume.

## On a VPS

1. A server with Docker and the Compose plugin, and a domain name you control.
2. DNS: an `A` record (and `AAAA` if the server has IPv6) of the domain pointing at the
   server's address. Wait until it resolves: Caddy asks Let's Encrypt for a certificate on
   the first request and the challenge fails when the name does not reach the server.
3. Firewall: allow only SSH, 80 and 443 (443 also over UDP for HTTP/3). With `ufw`:

   ```sh
   sudo ufw allow OpenSSH
   sudo ufw allow 80/tcp
   sudo ufw allow 443
   sudo ufw enable
   ```

   Docker publishes ports by editing the firewall rules itself, so a port you add to a compose
   file is open even if `ufw` does not list it. Publish nothing but what `compose.prod.yaml` already does.
4. Configuration: `cp .env.example .env` and set `APP_KEY` (`openssl rand -base64 32`) and
   mail (`MAIL_MAILER=resend`, `MAIL_FROM`, `RESEND_API_KEY`: without them mail only goes to
   the log). For `APP_EDITION=saas` also set the three `STRIPE_*` values. Keep `.env` out of
   git and readable only by you (`chmod 600 .env`). `compose.prod.yaml` forces
   `APP_ENV=production` and `APP_DEBUG=false`.
5. Start:

   ```sh
   export APP_DOMAIN=finance.example.com
   docker compose -f compose.prod.yaml up -d --build --wait
   docker compose -f compose.prod.yaml logs app | grep "setup code"
   ```

   Open `https://finance.example.com/setup` and create the owner with that code. Export
   `APP_DOMAIN` (or put it in `.env`) in every later `docker compose -f compose.prod.yaml`
   command: the file refuses to run without it.

Caddy sends `X-Forwarded-For` and `X-Forwarded-Proto`; the app trusts them only from the
compose network (`TRUSTED_PROXIES=172.28.0.0/24`), so the rate limits count each visitor by
their real address and a forged header from the internet changes nothing.
`bash scripts/docker-clientip-test.sh` checks this on a running stack.

### Why one replica

SQLite allows one writer. The app runs as a single container with the database in WAL mode and
a 5 second busy timeout, which is plenty for a household or a small team. A second replica, or
a tool that copies the live file, would break that guarantee: never run `docker compose up
--scale app=2`, and never copy `database.sqlite` while the app runs; use the backup command below.

## Backup

`db:backup` is a command of the app. It writes a consistent snapshot with `VACUUM INTO` while the
app keeps running, to a path that does not exist yet:

```sh
docker compose -f compose.prod.yaml exec app /app/app db:backup -to /app/storage/backups/$(date +%F).sqlite
```

(For the local stack leave out `-f compose.prod.yaml`.) The snapshot is a normal SQLite file. It
lives in the same volume as the database, so also copy it off the server:

```sh
docker compose -f compose.prod.yaml cp app:/app/storage/backups/2026-11-21.sqlite ./
```

A daily cron entry on the host does both; run it from the project directory and keep the copies
somewhere that is not this server. Old snapshots are not removed for you.

A snapshot keeps the data as it was when it was taken. Deleting a workspace or an account in the app never reaches a
snapshot: what was deleted is retained until you rotate (delete) the snapshot that holds it. Decide how long you keep them,
and tell the people whose data you host.

On Windows with Git Bash, put `MSYS_NO_PATHCONV=1` before the command: Git Bash otherwise turns
`/app/...` into a Windows path.

## Restore

The restore must go into a directory other than the snapshot's, and never over the live file
while the app runs:

```sh
C="docker compose -f compose.prod.yaml"
$C stop app
$C run --rm --no-deps --entrypoint /app/app app \
  db:backup -restore -from /app/storage/backups/2026-11-21.sqlite -to /app/storage/restore/database.sqlite
$C run --rm --no-deps --entrypoint rm app -f /app/storage/database.sqlite-wal /app/storage/database.sqlite-shm
$C run --rm --no-deps --entrypoint mv app /app/storage/restore/database.sqlite /app/storage/database.sqlite
$C up -d --wait app
```

The second command removes the leftover write-ahead files of the old database so SQLite does not
replay them over the restored file. To restore a snapshot that is not in the volume yet, copy it
in first with `docker compose -f compose.prod.yaml cp ./snapshot.sqlite app:/app/storage/backups/`
(the app container does not need to run for that, only to exist).

The image has no `sqlite3` binary on purpose. After a restore, open the app and check your data.

## Update to a new version

Releases are tagged `vX.Y.Z` and the image is published as `ghcr.io/alexgalhardo/vukem-finances:X.Y.Z`
(also on Docker Hub as `alexgalhardo/vukem-finances`). Take a backup first, then pick one way:

```sh
# From the published image:
export APP_IMAGE=ghcr.io/alexgalhardo/vukem-finances:0.2.0
docker compose -f compose.prod.yaml pull app
docker compose -f compose.prod.yaml up -d --no-build --wait

# Or build the tag yourself:
git fetch --tags && git checkout v0.2.0
VERSION=0.2.0 docker compose -f compose.prod.yaml up -d --build --wait
```

The app applies new migrations when it starts (`/app/app migrate` runs before the server).
`docker compose -f compose.prod.yaml exec app /app/app about` prints the running version.
To go back, restore the backup you took and start the older tag: migrations do not run backwards.

## Locked out of the owner account

Sign-up is closed in `personal`, and the reset-by-email flow needs working mail. From the server:

```sh
docker compose -f compose.prod.yaml exec app /app/app user:reset-password -email you@example.com
```

It prints a new random password once and ends every session of that user. Sign in with it and
change it in the account settings.

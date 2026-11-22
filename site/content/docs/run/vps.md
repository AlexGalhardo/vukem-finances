---
title: "A VPS with Caddy"
description: "Run vukem-finances on your own server with compose.prod.yaml: Caddy gets the HTTPS certificate and the app publishes no port."
---

`compose.prod.yaml` is the stack for a server. Caddy terminates HTTPS with an automatic certificate and forwards to the app, which publishes no port of its own. This is also how an operator runs the SaaS edition; no hosted service is offered, so the operator is you.

## What you need

1. A server with Docker and the Compose plugin, and a domain name you control.
2. An `A` record (and `AAAA` if the server has IPv6) of the domain pointing at the server. Wait until it resolves: Caddy asks Let's Encrypt for a certificate on the first request, and the challenge fails when the name does not reach the server.
3. A firewall that allows only SSH, 80 and 443 (443 also over UDP for HTTP/3):

```sh
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443
sudo ufw enable
```

Docker edits the firewall rules by itself when it publishes a port, so a port you add to a compose file is open even if `ufw` does not list it. Publish nothing but what `compose.prod.yaml` already does.

## Configure

Copy `.env.example` to `.env` and set `APP_KEY` (`openssl rand -base64 32`). For real email set `MAIL_MAILER=resend`, `MAIL_FROM` and `RESEND_API_KEY`; without them mail only goes to the log. For `APP_EDITION=saas` set the three `STRIPE_*` values and `APP_URL`. Keep `.env` out of git and readable only by you (`chmod 600 .env`). `compose.prod.yaml` forces `APP_ENV=production` and `APP_DEBUG=false`.

## Start

```sh
export APP_DOMAIN=finance.example.com
docker compose -f compose.prod.yaml up -d --build --wait
docker compose -f compose.prod.yaml logs app | grep "setup code"
```

Open `https://finance.example.com/setup` and create the owner with that code. Export `APP_DOMAIN`, or put it in `.env`, for every later `docker compose -f compose.prod.yaml` command: the file refuses to run without it.

## Real client addresses

Caddy sends `X-Forwarded-For` and `X-Forwarded-Proto`. The app trusts them only from the compose network (`TRUSTED_PROXIES` is the compose subnet), so rate limits count each visitor by their real address and a forged header from the internet changes nothing.

## Why one replica

SQLite allows one writer. The app runs as a single container with the database in WAL mode and a 5 second busy timeout, which is plenty for a household or a small team. Never run `docker compose up --scale app=2`, and never copy `database.sqlite` while the app runs: use the [backup command](/docs/run/backup-restore/).

## Locked out of the owner account

Sign-up is closed in the personal edition and reset by email needs working mail. From the server:

```sh
docker compose -f compose.prod.yaml exec app /app/app user:reset-password -email you@example.com
```

It prints a new random password once and ends every session of that user.

## Next steps

[Upgrade](/docs/run/upgrading/) when a release comes out, and set up [backups](/docs/run/backup-restore/) on a schedule. The full guide is `deploy/README.md`.

---
title: "Mobile sync and pairing"
description: "Pair the phone with a server using a short code, how a sync run works, how conflicts are resolved and what stays on the phone."
---

The [mobile app](/docs/run/mobile/) works alone. Sync is optional and connects the phone to your personal server, or to a SaaS installation, so both show the same data.

## Pair a phone

1. On the web, open Devices and press Add device. The page shows an 8 character code, valid once and for 10 minutes, and the server address.
2. On the phone open More, Sync, Connect, and type the address, the code and a name for the phone.
3. The server answers with a token for that workspace, with only the `sync` scope and a life of 365 days. It appears on the Tokens page as a phone. The phone keeps it in its secure store only.

The address must be `https`. Plain `http` is accepted only for `localhost`, private network addresses and `.local` names, after a confirmation that the traffic is not encrypted. Plain `http` to a public host is refused. For a server on your own network, prefer the Caddy setup of [the VPS guide](/docs/run/vps/).

You can revoke a device on the Devices page. The phone then shows that sign-in is needed, keeps its data and offers to pair again.

## What a run does

A run pushes up to 200 queued changes in the order you made them, each with a fixed id so a retry after a lost response never writes twice, then pulls what changed on the server page by page, then checks integrity. It runs on Sync now, when the app comes to the foreground, 3 seconds after a local write and every 5 minutes while the app is open. A failure backs off and retries. If the server says the phone's cursor is too old, the phone downloads a snapshot and replaces its copy.

## Conflicts

- A pending edit on the phone survives a pull of the same row.
- A delete wins over a concurrent edit.
- When the same field changes on both sides, the later arrival wins and the loser is recorded.

Sync, Review conflicts shows each lost value with both sides and lets you apply it again as a normal edit, or dismiss it. A change the server refuses for good is removed from the queue and listed in the problems screen.

## The first connection

A phone with no data connects without a prompt. A phone with data asks you to Merge (keep both sides: an account, category or tag with the same name and kind as a server row takes the server's id), Replace the phone's data with the server's after a confirmation and an offered export, or Cancel.

## Limits

- Sync runs only while the app is open: there is no background sync.
- One workspace per phone.
- Goals, notifications and AI data are not in the app and are not synced.
- There is no end-to-end encryption.
- The server limits a device to 30 pushes and 60 pulls a minute.

The wire protocol is in [the sync protocol](/docs/integrate/sync-protocol/).

import { Head, Link, router, useForm, usePage } from "@inertiajs/react";
import { type FormEvent, useState } from "react";

import Field from "../components/Field";
import { Card, Select, danger, primary } from "../components/ui";
import type { Page, SharedProps } from "../page-props";

type InviteLink = { url: string; expiresAt: string };

// The invite link is shown once, right after the invite is created: a
// personal installation usually sends no email, and only a hash is stored.
function InviteLinkNotice({ link }: { link: InviteLink }) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    await navigator.clipboard.writeText(link.url);
    setCopied(true);
  };

  return (
    <Card title="Invitation link">
      <p className="text-sm">
        Send this link to the person you invited. It is a secret: anyone who has it can join this workspace until it
        expires on {new Date(link.expiresAt).toLocaleString()}. It will not be shown again.
      </p>
      <div className="mt-3 flex flex-wrap items-center gap-3">
        <input readOnly aria-label="Invitation link" value={link.url} className="min-w-0 flex-1 rounded-lg border bg-background px-3 py-2 text-sm" onFocus={(e) => e.currentTarget.select()} />
        <button type="button" onClick={copy} className={primary}>
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
    </Card>
  );
}

export default function Workspace({ workspaceDetails, members, invites, limits, canManage, audit }: Page<"Workspace">) {
  const page = usePage<SharedProps>();
  const inviteLink = page.flash?.inviteLink as InviteLink | undefined;
  const personal = page.props.workspace?.edition === "personal";
  const role = page.props.workspace?.role;
  const isOwner = role === "owner";
  const settings = useForm({ name: workspaceDetails.name, currency: workspaceDetails.currency, timezone: workspaceDetails.timezone });
  const invite = useForm({ email: "", role: "member" });
  const create = useForm({ name: "", currency: "USD" });

  const save = (e: FormEvent) => {
    e.preventDefault();
    settings.put("/workspace");
  };

  const send = (e: FormEvent) => {
    e.preventDefault();
    invite.post("/workspace/invites", { onSuccess: () => invite.reset("email") });
  };

  const add = (e: FormEvent) => {
    e.preventDefault();
    create.post("/workspaces", { onSuccess: () => create.reset("name") });
  };

  return (
    <>
      <Head title="Workspace" />
      <h1 className="text-2xl font-semibold">{workspaceDetails.name}</h1>
      <Card title="Settings">
        <form onSubmit={save} className="grid gap-3 sm:grid-cols-4">
          <Field label="Name" required disabled={!canManage} value={settings.data.name} onChange={(e) => settings.setData("name", e.target.value)} error={settings.errors.name} />
          <Field label="Currency (ISO 4217)" required maxLength={3} disabled={!canManage} value={settings.data.currency} onChange={(e) => settings.setData("currency", e.target.value.toUpperCase())} error={settings.errors.currency} />
          <Field label="Timezone (IANA, such as America/Sao_Paulo)" required disabled={!canManage} value={settings.data.timezone} onChange={(e) => settings.setData("timezone", e.target.value)} error={settings.errors.timezone} />
          {canManage && (
            <div className="self-end">
              <button type="submit" disabled={settings.processing} className={primary}>
                Save
              </button>
            </div>
          )}
        </form>
      </Card>
      <Card title="Members">
        <ul className="divide-y divide-zinc-200 dark:divide-zinc-800">
          {members.map((m) => (
            <li key={m.userId} className="flex items-center justify-between py-2">
              <span>
                {m.name} <span className="text-sm text-zinc-500">{m.email}, {m.role}</span>
              </span>
              <span className="flex items-center gap-3">
                {canManage && m.role !== "owner" && (
                  <select
                    aria-label={`Role of ${m.name}`}
                    value={m.role}
                    onChange={(e) => router.patch(`/workspace/members/${m.userId}`, { role: e.target.value })}
                    className="rounded-lg border bg-background px-2 py-1 text-sm"
                  >
                    <option value="admin">Admin</option>
                    <option value="member">Member</option>
                    <option value="viewer">Viewer</option>
                  </select>
                )}
                {isOwner && m.role !== "owner" && (
                  <button
                    type="button"
                    className={danger}
                    onClick={() => {
                      if (confirm(`Make ${m.name} the owner? You become an admin.`)) {
                        router.post("/workspace/owner", { userId: m.userId });
                      }
                    }}
                  >
                    Make owner
                  </button>
                )}
                {canManage && m.role !== "owner" && (
                  <Link href={`/workspace/members/${m.userId}`} method="delete" as="button" className={danger}>
                    Remove
                  </Link>
                )}
              </span>
            </li>
          ))}
          {invites.map((i) => (
            <li key={i.id} className="py-2 text-sm text-zinc-500">
              {i.email} ({i.role}): invitation pending
            </li>
          ))}
        </ul>
        {role && !isOwner && (
          <Link href="/workspace/leave" method="post" as="button" className={`${danger} mt-3`}>
            Leave this workspace
          </Link>
        )}
        {!personal && limits.members >= 0 && (
          <p className="mt-3 text-sm text-zinc-500">
            The {limits.plan} plan allows {limits.members} member.{" "}
            <Link href="/billing" className="text-indigo-600 hover:underline dark:text-indigo-400">
              Upgrade
            </Link>{" "}
            to invite people.
          </p>
        )}
      </Card>
      {inviteLink && <InviteLinkNotice link={inviteLink} />}
      {canManage && (
        <Card title="Invite someone">
          <form onSubmit={send} className="grid gap-3 sm:grid-cols-3">
            <Field label="Email" type="email" required value={invite.data.email} onChange={(e) => invite.setData("email", e.target.value)} error={invite.errors.email} />
            <Select label="Role" value={invite.data.role} onChange={(e) => invite.setData("role", e.target.value)} error={invite.errors.role}>
              <option value="member">Member: records transactions</option>
              <option value="admin">Admin: also manages data and people</option>
              <option value="viewer">Viewer: sees everything, changes nothing</option>
            </Select>
            <div className="self-end">
              <button type="submit" disabled={invite.processing} className={primary}>
                Send invitation
              </button>
            </div>
          </form>
        </Card>
      )}
      {canManage && (
        <Card title="Activity">
          {audit.length === 0 ? (
            <p className="text-sm text-zinc-500">Nothing yet.</p>
          ) : (
            <ul className="divide-y divide-zinc-200 text-sm dark:divide-zinc-800">
              {audit.map((e) => (
                <li key={e.id} className="py-2">
                  <span className="font-medium">{e.actor}</span> {e.action} <span className="text-zinc-500">{e.subject}</span>
                  <span className="float-right text-zinc-500">{e.createdAt}</span>
                </li>
              ))}
            </ul>
          )}
        </Card>
      )}
      {!personal && (
        <Card title="Create another workspace">
          <form onSubmit={add} className="grid gap-3 sm:grid-cols-3">
            <Field label="Name" required value={create.data.name} onChange={(e) => create.setData("name", e.target.value)} error={create.errors.name} />
            <Field label="Currency (ISO 4217)" required maxLength={3} value={create.data.currency} onChange={(e) => create.setData("currency", e.target.value.toUpperCase())} error={create.errors.currency} />
            <div className="self-end">
              <button type="submit" disabled={create.processing} className={primary}>
                Create
              </button>
            </div>
          </form>
        </Card>
      )}
    </>
  );
}

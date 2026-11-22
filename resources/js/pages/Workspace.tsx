import { Head, Link, router, useForm, usePage } from "@inertiajs/react";
import { Download } from "lucide-react";
import { type FormEvent, useState } from "react";
import Field from "../components/Field";
import PageHeader from "../components/PageHeader";
import SettingsSection from "../components/settings/SettingsSection";
import { Select } from "../components/ui";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { NativeSelect } from "../components/ui/native-select";
import { HelpHint } from "../help/HelpHint";
import type { Key } from "../i18n/keys";
import { canExportJson, workspaceJsonUrl } from "../lib/exports";
import { useFinanceI18n } from "../lib/i18n";
import { fill } from "../lib/pix";
import type { Page, SharedProps } from "../page-props";

type InviteLinkProp = { url: string; expiresAt: string };

const roleKeys: Record<string, Key> = {
  owner: "wsRoleOwner",
  admin: "wsRoleAdmin",
  member: "wsRoleMember",
  viewer: "wsRoleViewer",
};

// The invite link is shown once, right after the invite is created: a
// personal installation usually sends no email, and only a hash is stored.
function InviteLinkNotice({ link }: { link: InviteLinkProp }) {
  const { t } = useFinanceI18n();
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    await navigator.clipboard.writeText(link.url);
    setCopied(true);
  };

  return (
    <SettingsSection title={t("wsLinkTitle")}>
      <p className="text-body text-pretty">
        {fill(t("wsLinkBody"), { date: new Date(link.expiresAt).toLocaleString() })}
      </p>
      <div className="flex flex-wrap items-center gap-3">
        <Input
          readOnly
          aria-label={t("wsLinkTitle")}
          value={link.url}
          className="min-w-0 flex-1"
          onFocus={(e) => e.currentTarget.select()}
        />
        <Button type="button" onClick={copy}>
          {copied ? t("wsLinkCopied") : t("wsLinkCopy")}
        </Button>
      </div>
    </SettingsSection>
  );
}

export default function Workspace({ workspaceDetails, members, invites, limits, canManage, audit }: Page<"Workspace">) {
  const page = usePage<SharedProps>();
  const { t } = useFinanceI18n();
  const inviteLink = page.flash?.inviteLink as InviteLinkProp | undefined;
  const personal = page.props.workspace?.edition === "personal";
  const role = page.props.workspace?.role;
  const isOwner = role === "owner";
  const [ownerCandidate, setOwnerCandidate] = useState<{ userId: number; name: string } | null>(null);
  const settings = useForm({
    name: workspaceDetails.name,
    currency: workspaceDetails.currency,
    timezone: workspaceDetails.timezone,
  });
  const invite = useForm({ email: "", role: "member" });
  const create = useForm({ name: "", currency: "USD" });
  const roleLabel = (value: string) => t(roleKeys[value] ?? "wsRoleMember");

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
      <Head title={t("workspace")} />
      <PageHeader title={workspaceDetails.name} />
      <div className="flex flex-col">
        <SettingsSection title={t("wsSettingsTitle")} description={t("wsSettingsDescription")}>
          <form onSubmit={save} className="flex flex-col gap-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <Field
                label={t("wsName")}
                required
                disabled={!canManage}
                value={settings.data.name}
                onChange={(e) => settings.setData("name", e.target.value)}
                error={settings.errors.name}
              />
              <Field
                label={t("wsCurrency")}
                required
                maxLength={3}
                disabled={!canManage}
                value={settings.data.currency}
                onChange={(e) => settings.setData("currency", e.target.value.toUpperCase())}
                error={settings.errors.currency}
              />
              <div className="sm:col-span-2">
                <Field
                  label={t("wsTimezone")}
                  required
                  disabled={!canManage}
                  value={settings.data.timezone}
                  onChange={(e) => settings.setData("timezone", e.target.value)}
                  error={settings.errors.timezone}
                />
              </div>
            </div>
            {canManage ? (
              <div>
                <Button type="submit" loading={settings.processing}>
                  {t("wsSave")}
                </Button>
              </div>
            ) : (
              <p className="text-caption text-muted-foreground">{t("wsAdminOnly")}</p>
            )}
          </form>
        </SettingsSection>

        <SettingsSection
          title={t("wsMembersTitle")}
          description={t("wsMembersDescription")}
          tour="workspace-members"
          help={<HelpHint id="hintWsRoles" labelKey="wsHelpRoles" tutorialStep="Workspace:0" />}
        >
          <ul className="flex flex-col divide-y divide-border">
            {members.map((m) => (
              <li key={m.userId} className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 py-3">
                <div className="flex min-w-0 flex-col gap-0.5">
                  <span className="text-label">{m.name}</span>
                  <span className="text-caption break-all text-muted-foreground">{m.email}</span>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  {canManage && m.role !== "owner" ? (
                    <NativeSelect
                      aria-label={fill(t("wsRoleOf"), { name: m.name })}
                      value={m.role}
                      onChange={(e) => router.patch(`/workspace/members/${m.userId}`, { role: e.target.value })}
                    >
                      <option value="admin">{t("wsRoleAdmin")}</option>
                      <option value="member">{t("wsRoleMember")}</option>
                      <option value="viewer">{t("wsRoleViewer")}</option>
                    </NativeSelect>
                  ) : (
                    <Badge variant={m.role === "owner" ? "success" : "neutral"}>{roleLabel(m.role)}</Badge>
                  )}
                  {isOwner && m.role !== "owner" && (
                    <>
                      <Button
                        type="button"
                        variant="destructive-outline"
                        size="sm"
                        onClick={() => setOwnerCandidate({ userId: m.userId, name: m.name })}
                      >
                        {t("wsMakeOwner")}
                      </Button>
                      <HelpHint id="hintWsTransferOwnership" labelKey="wsHelpTransfer" side="left" />
                    </>
                  )}
                  {canManage && m.role !== "owner" && (
                    <Button asChild variant="destructive-outline" size="sm">
                      <Link href={`/workspace/members/${m.userId}`} method="delete" as="button">
                        {t("wsRemove")}
                      </Link>
                    </Button>
                  )}
                </div>
              </li>
            ))}
            {invites.map((i) => (
              <li key={i.id} className="py-3 text-body text-muted-foreground">
                {fill(t("wsPending"), { email: i.email, role: roleLabel(i.role) })}
              </li>
            ))}
          </ul>
          {!canManage && <p className="text-caption text-muted-foreground">{t("wsMembersAdminOnly")}</p>}
          {canManage && !isOwner && <p className="text-caption text-muted-foreground">{t("wsOwnerOnlyTransfer")}</p>}
          {role && !isOwner && (
            <div>
              <Button asChild variant="destructive-outline">
                <Link href="/workspace/leave" method="post" as="button">
                  {t("wsLeave")}
                </Link>
              </Button>
            </div>
          )}
          {!personal && limits.members >= 0 && (
            <p className="text-body text-muted-foreground">
              {fill(t("wsPlanLimit"), { plan: limits.plan, count: limits.members })}{" "}
              <Link href="/billing" className="text-brand-text underline underline-offset-4">
                {t("wsUpgrade")}
              </Link>
              .
            </p>
          )}
        </SettingsSection>

        {inviteLink && <InviteLinkNotice link={inviteLink} />}

        {canManage && (
          <SettingsSection title={t("wsInviteTitle")} description={t("wsInviteDescription")} tour="workspace-invite">
            <form onSubmit={send} className="flex flex-col gap-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <Field
                  label={t("wsEmail")}
                  type="email"
                  required
                  value={invite.data.email}
                  onChange={(e) => invite.setData("email", e.target.value)}
                  error={invite.errors.email}
                />
                <Select
                  label={t("wsRole")}
                  value={invite.data.role}
                  onChange={(e) => invite.setData("role", e.target.value)}
                  error={invite.errors.role}
                >
                  <option value="member">{t("wsRoleMemberOption")}</option>
                  <option value="admin">{t("wsRoleAdminOption")}</option>
                  <option value="viewer">{t("wsRoleViewerOption")}</option>
                </Select>
              </div>
              <div>
                <Button type="submit" loading={invite.processing}>
                  {t("wsInviteSend")}
                </Button>
              </div>
            </form>
          </SettingsSection>
        )}

        {canManage && (
          <SettingsSection title={t("wsActivityTitle")} description={t("wsActivityDescription")}>
            {audit.length === 0 ? (
              <p className="text-body text-muted-foreground">{t("wsActivityEmpty")}</p>
            ) : (
              <ul className="flex flex-col divide-y divide-border text-body">
                {audit.map((e) => (
                  <li key={e.id} className="flex flex-wrap justify-between gap-x-4 gap-y-1 py-2">
                    <span className="min-w-0">
                      <span className="text-label">{e.actor}</span> {e.action}{" "}
                      <span className="text-muted-foreground">{e.subject}</span>
                    </span>
                    <span className="text-caption text-muted-foreground tabular-nums">{e.createdAt}</span>
                  </li>
                ))}
              </ul>
            )}
          </SettingsSection>
        )}

        {canExportJson(role) && (
          <SettingsSection title={t("wsDataTitle")} description={t("wsDataDescription")} tour="workspace-export">
            <div>
              <Button asChild variant="outline">
                <a href={workspaceJsonUrl} download>
                  <Download aria-hidden="true" data-icon="inline-start" />
                  {t("wsDataExport")}
                </a>
              </Button>
            </div>
          </SettingsSection>
        )}

        {!personal && (
          <SettingsSection title={t("wsCreateTitle")} description={t("wsCreateDescription")}>
            <form onSubmit={add} className="flex flex-col gap-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <Field
                  label={t("wsName")}
                  required
                  value={create.data.name}
                  onChange={(e) => create.setData("name", e.target.value)}
                  error={create.errors.name}
                />
                <Field
                  label={t("wsCurrency")}
                  required
                  maxLength={3}
                  value={create.data.currency}
                  onChange={(e) => create.setData("currency", e.target.value.toUpperCase())}
                  error={create.errors.currency}
                />
              </div>
              <div>
                <Button type="submit" loading={create.processing}>
                  {t("wsCreate")}
                </Button>
              </div>
            </form>
          </SettingsSection>
        )}
      </div>

      <Dialog open={ownerCandidate !== null} onOpenChange={(open) => !open && setOwnerCandidate(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{fill(t("wsMakeOwnerTitle"), { name: ownerCandidate?.name ?? "" })}</DialogTitle>
            <DialogDescription>{t("wsMakeOwnerBody")}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOwnerCandidate(null)}>
              {t("wsCancel")}
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={() => {
                if (ownerCandidate) router.post("/workspace/owner", { userId: ownerCandidate.userId });
                setOwnerCandidate(null);
              }}
            >
              {t("wsMakeOwner")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

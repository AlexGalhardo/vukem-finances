import { router } from "@inertiajs/react";
import { type ReactNode, useRef, useState } from "react";

import { useFinanceI18n } from "../lib/i18n";
import { Button } from "./ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "./ui/dialog";
import { Input } from "./ui/input";
import { Label } from "./ui/label";

export interface ConfirmDeleteProps {
  href: string;
  name: string;
  trigger: ReactNode;
  /** For the worst cases: the confirm button stays off until the person types the name. */
  requireName?: boolean;
}

/**
 * Asks before a destructive request: the trigger opens a dialog and only its confirm button sends the DELETE.
 * Focus starts on the safe action, Cancel, so Enter never deletes by accident.
 */
export default function ConfirmDelete({ href, name, trigger, requireName = false }: ConfirmDeleteProps) {
  const { t } = useFinanceI18n();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [typed, setTyped] = useState("");
  const cancel = useRef<HTMLButtonElement>(null);
  const armed = !requireName || typed.trim() === name;

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) setTyped("");
      }}
    >
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent
        onOpenAutoFocus={(event) => {
          event.preventDefault();
          cancel.current?.focus();
        }}
      >
        <DialogHeader>
          <DialogTitle>{t("confirmDeleteTitle")}</DialogTitle>
          <DialogDescription>
            {name}. {t("confirmDeleteBody")}
          </DialogDescription>
        </DialogHeader>
        {requireName && (
          <div className="flex flex-col gap-2">
            <Label htmlFor="confirm-delete-name">
              {t("confirmTypeName")} <strong>{name}</strong>
            </Label>
            <Input
              id="confirm-delete-name"
              value={typed}
              autoComplete="off"
              spellCheck={false}
              onChange={(event) => setTyped(event.target.value)}
            />
          </div>
        )}
        <DialogFooter>
          <Button ref={cancel} type="button" variant="outline" onClick={() => setOpen(false)}>
            {t("cancel")}
          </Button>
          <Button
            type="button"
            variant="destructive"
            loading={busy}
            disabled={!armed}
            onClick={() =>
              router.delete(href, {
                preserveScroll: true,
                onStart: () => setBusy(true),
                onFinish: () => {
                  setBusy(false);
                  setOpen(false);
                },
              })
            }
          >
            {t("delete")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

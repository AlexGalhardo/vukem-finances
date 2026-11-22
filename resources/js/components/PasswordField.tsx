import { CircleAlert, Eye, EyeOff } from "lucide-react";
import { useId, useState } from "react";

import { useFinanceI18n } from "../lib/i18n";
import type { PasswordAdvice, PasswordCheck } from "../lib/password-policy";
import { cn } from "../lib/utils";
import { Input } from "./ui/input";

interface Props {
  label: string;
  // The server's field name ("password", "password_confirmation"): it is what
  // a password manager reads. The DOM id comes from useId, so two fields with
  // the same name on one page still get distinct ids.
  name: string;
  value: string;
  onChange: (value: string) => void;
  autoComplete: "new-password" | "current-password";
  error?: string;
  // The server rules this value must satisfy, from lib/password-policy.
  // Omit them on a login form: it must not advertise the policy.
  checks?: readonly PasswordCheck[];
  // Optional strength rating. Kept visually apart from `checks` because it is
  // advice: the server accepts anything `checks` accepts, whatever this says.
  advice?: PasswordAdvice;
  /** `lg` is the 44 px field of the auth screens. */
  inputSize?: "default" | "lg";
}

// Every step also has its text, so the meter is never the only signal.
const meter = ["bg-muted", "bg-expense", "bg-warning", "bg-info", "bg-brand"];

// PasswordField is a password input with a show/hide toggle inside the field
// and, when `checks` are given, the server's rules checked as you type.
//
// The label is bound with htmlFor instead of wrapping the input: a <label> that
// encloses the toggle button folds the button's text into the field's
// accessible name, and the field announces as "Password Show password".
//
// Visibility lives in component state only, so a revealed password is hidden
// again on the next mount; revealing it is a deliberate act every time.
//
// The checklist carries role="list" explicitly: Tailwind's reset sets
// list-style: none, which drops the list role in some screen readers. Each
// rule's state is in its text too, never in its colour alone.
export default function PasswordField({
  label,
  name,
  value,
  onChange,
  autoComplete,
  error,
  checks,
  advice,
  inputSize,
}: Props) {
  const { t } = useFinanceI18n();
  const [visible, setVisible] = useState(false);
  const id = useId();
  const errorId = `${id}-error`;
  const checksId = `${id}-checks`;
  const adviceId = `${id}-advice`;

  const describedBy = [error && errorId, checks && checksId, advice && adviceId].filter(Boolean).join(" ");

  return (
    <div className="flex flex-col gap-2">
      <label htmlFor={id} className="text-label">
        {label}
      </label>
      <div className="relative">
        <Input
          id={id}
          name={name}
          type={visible ? "text" : "password"}
          autoComplete={autoComplete}
          required
          value={value}
          onChange={(e) => onChange(e.target.value)}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy || undefined}
          inputSize={inputSize}
          className="pe-11"
        />
        <button
          type="button"
          onClick={() => setVisible((shown) => !shown)}
          aria-label={visible ? t("hidePassword") : t("showPassword")}
          aria-pressed={visible}
          aria-controls={id}
          className="absolute inset-y-0 end-0 flex w-11 items-center justify-center rounded-e-md text-muted-foreground transition-colors hover:text-foreground"
        >
          {visible ? <EyeOff aria-hidden="true" className="size-4" /> : <Eye aria-hidden="true" className="size-4" />}
        </button>
      </div>
      {error && (
        <p id={errorId} className="flex items-start gap-1.5 text-caption text-expense">
          <CircleAlert aria-hidden="true" className="mt-px size-3.5 shrink-0" />
          {error}
        </p>
      )}
      {checks && (
        // biome-ignore lint/a11y/noRedundantRoles: Safari/VoiceOver drops list semantics from a list with list-style: none
        <ul id={checksId} role="list" className="space-y-1 text-caption">
          {checks.map((check) => (
            <li key={check.id} className={check.met ? "text-brand-text" : "text-muted-foreground"}>
              <span aria-hidden="true" className="mr-1.5 inline-block w-3 text-center">
                {check.met ? "✓" : "•"}
              </span>
              {check.label}
              <span className="sr-only"> ({check.met ? t("passwordMet") : t("passwordNotMet")})</span>
            </li>
          ))}
        </ul>
      )}
      {advice && (
        <div id={adviceId} className="text-caption text-muted-foreground">
          <div aria-hidden="true" className="mb-1 flex gap-1">
            {[1, 2, 3, 4].map((step) => (
              <span
                key={step}
                className={cn("h-1 flex-1 rounded-full", step <= advice.score ? meter[advice.score] : meter[0])}
              />
            ))}
          </div>
          {t("passwordStrength")}: {advice.label} ({t("passwordAdviceNote")}). {advice.hint}
        </div>
      )}
    </div>
  );
}

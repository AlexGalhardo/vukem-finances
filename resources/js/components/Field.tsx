import { CircleAlert } from "lucide-react";
import { type InputHTMLAttributes, useId } from "react";
import { Input } from "./ui/input";

interface Props extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
}

/** A labelled input with its validation error: an icon and a message below, linked by aria-describedby. */
export default function Field({ label, error, id, ...input }: Props) {
  const generated = useId();
  const fieldId = id ?? generated;
  const errorId = `${fieldId}-error`;
  return (
    <div className="flex flex-col gap-2">
      <label htmlFor={fieldId} className="text-label">
        {label}
      </label>
      <Input
        {...input}
        id={fieldId}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? errorId : input["aria-describedby"]}
      />
      {error && (
        <p id={errorId} className="flex items-start gap-1.5 text-caption text-expense">
          <CircleAlert aria-hidden="true" className="mt-px size-3.5 shrink-0" />
          {error}
        </p>
      )}
    </div>
  );
}

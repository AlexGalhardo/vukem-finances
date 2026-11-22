import type { ReactNode, SelectHTMLAttributes } from "react";
import { useId } from "react";
import { buttonVariants } from "./ui/button";
import { CardContent, CardHeader, CardTitle, Card as ShadcnCard } from "./ui/card";
import { Field, FieldError, FieldLabel } from "./ui/field";
import { inputVariants } from "./ui/input";
import { NativeSelect } from "./ui/native-select";
export const primary = buttonVariants();
export const secondary = buttonVariants({ variant: "outline" });
export const danger = buttonVariants({ variant: "destructive-outline", size: "sm" });
export const input = inputVariants();
export function Card({ title, children }: { title?: string; children: ReactNode }) {
  return (
    <ShadcnCard>
      {title && (
        <CardHeader>
          <CardTitle>{title}</CardTitle>
        </CardHeader>
      )}
      <CardContent>{children}</CardContent>
    </ShadcnCard>
  );
}
interface SelectProps extends Omit<SelectHTMLAttributes<HTMLSelectElement>, "size"> {
  label: string;
  error?: string;
}
export function Select({ label, error, children, id, ...rest }: SelectProps) {
  const generated = useId();
  const fieldId = id ?? generated;
  return (
    <Field data-invalid={!!error}>
      <FieldLabel htmlFor={fieldId}>{label}</FieldLabel>
      <NativeSelect {...rest} id={fieldId} aria-invalid={!!error}>
        {children}
      </NativeSelect>
      <FieldError>{error}</FieldError>
    </Field>
  );
}

import { type RefObject, useEffect, useRef } from "react";

/**
 * Moves focus to the first field the server marked invalid (`aria-invalid="true"`) each time a submit comes back with
 * errors, so a keyboard or screen reader user lands on the problem. When the error belongs to no field, the error
 * summary takes the focus instead. Put the returned ref on the form.
 */
export function useFocusFirstError(errors: Record<string, string | undefined>): RefObject<HTMLFormElement | null> {
  const formRef = useRef<HTMLFormElement | null>(null);
  useEffect(() => {
    if (Object.keys(errors).length === 0) return;
    const form = formRef.current;
    if (!form) return;
    const field = form.querySelector<HTMLElement>('[aria-invalid="true"]');
    if (field) {
      field.focus();
      return;
    }
    form.parentElement?.querySelector<HTMLElement>('[data-slot="auth-error-summary"]')?.focus();
  }, [errors]);
  return formRef;
}

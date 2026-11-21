import { X } from "lucide-react";
import { type KeyboardEvent, useId, useState } from "react";
import { useFinanceI18n } from "../../lib/i18n";
import { Badge } from "../ui/badge";
import { Field, FieldDescription, FieldError, FieldLabel } from "../ui/field";
import { Input } from "../ui/input";
import { addTag, MAX_TAGS } from "./entryForm";

export interface TagsFieldProps {
  tags: string[];
  onChange: (tags: string[]) => void;
  /** Names of the tags of the workspace, offered as suggestions while typing. */
  suggestions: readonly string[];
  error?: string;
}

/** Type a tag and press Enter (or comma) to add it; Backspace on an empty box takes the last one off. Tags that do not exist yet are created on save. */
export default function TagsField({ tags, onChange, suggestions, error }: TagsFieldProps) {
  const { t } = useFinanceI18n();
  const [draft, setDraft] = useState("");
  const [announcement, setAnnouncement] = useState("");
  const id = useId();
  const listId = `${id}-suggestions`;
  const helpId = `${id}-help`;

  function commit(): void {
    const next = addTag(tags, draft);
    if (next.length !== tags.length) {
      onChange(next);
      setAnnouncement(`${t("tagAdded")}: ${draft.trim()}`);
    } else if (draft.trim() !== "" && tags.length >= MAX_TAGS) {
      setAnnouncement(t("tagsFull"));
    }
    setDraft("");
  }

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>): void {
    // An Enter that confirms an input-method candidate must not add the half-typed tag.
    if (event.nativeEvent.isComposing) return;
    if (event.key === "Enter" || event.key === ",") {
      event.preventDefault();
      commit();
    } else if (event.key === "Backspace" && draft === "" && tags.length > 0) {
      const last = tags[tags.length - 1] ?? "";
      onChange(tags.slice(0, -1));
      setAnnouncement(`${t("tagRemoved")}: ${last}`);
    }
  }

  const unused = suggestions.filter((name) => !tags.some((tag) => tag.toLowerCase() === name.toLowerCase()));

  return (
    <Field data-invalid={!!error}>
      <FieldLabel htmlFor={id}>{t("tags")}</FieldLabel>
      {tags.length > 0 && (
        <ul className="flex flex-wrap gap-1.5" aria-label={t("tags")}>
          {tags.map((tag) => (
            <li key={tag.toLowerCase()}>
              <Badge variant="secondary" className="gap-1 pr-1">
                {tag}
                <button
                  type="button"
                  className="inline-flex size-5 items-center justify-center rounded-full hover:bg-background pointer-coarse:size-8"
                  aria-label={`${t("removeTag")} ${tag}`}
                  onClick={() => {
                    onChange(tags.filter((other) => other !== tag));
                    setAnnouncement(`${t("tagRemoved")}: ${tag}`);
                  }}
                >
                  <X aria-hidden="true" className="size-3" />
                </button>
              </Badge>
            </li>
          ))}
        </ul>
      )}
      <Input
        id={id}
        name="tag-draft"
        type="text"
        autoComplete="off"
        autoCapitalize="none"
        spellCheck={false}
        enterKeyHint="done"
        list={listId}
        maxLength={40}
        value={draft}
        disabled={tags.length >= MAX_TAGS}
        aria-describedby={helpId}
        aria-invalid={!!error}
        onChange={(event) => setDraft(event.target.value)}
        onKeyDown={onKeyDown}
        onBlur={commit}
      />
      <datalist id={listId}>
        {unused.map((name) => (
          <option key={name} value={name} />
        ))}
      </datalist>
      <FieldDescription id={helpId}>{t("tagsHelp")}</FieldDescription>
      <FieldError>{error}</FieldError>
      <div role="status" aria-live="polite" className="sr-only">
        {announcement}
      </div>
    </Field>
  );
}

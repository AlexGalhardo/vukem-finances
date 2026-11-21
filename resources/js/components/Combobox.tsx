import { type KeyboardEvent, type ReactNode, useCallback, useEffect, useId, useRef, useState } from "react";
import { cn } from "../lib/utils";

export interface ComboboxItem {
  id: string;
  label: string;
  group?: string;
  /** Right-aligned hint, such as the shortcut keys. */
  hint?: ReactNode;
  icon?: ReactNode;
}

export interface ComboboxProps {
  items: readonly ComboboxItem[];
  query: string;
  onQueryChange: (query: string) => void;
  onSelect: (item: ComboboxItem) => void;
  /** Accessible name of the input. */
  label: string;
  placeholder?: string;
  /** Says "N results" to screen readers; called whenever the list changes. */
  resultsLabel: (count: number) => string;
  /** The listbox is always shown (the command palette). Otherwise it opens on typing or ArrowDown. */
  alwaysOpen?: boolean;
  /** Called when focus leaves the input (the popup closes by itself). */
  onInputBlur?: () => void;
  className?: string;
  inputClassName?: string;
  listClassName?: string;
}

/** ARIA 1.2 combobox with a listbox popup: focus stays in the input, `aria-activedescendant` names the option. */
export default function Combobox({
  items,
  query,
  onQueryChange,
  onSelect,
  label,
  placeholder,
  resultsLabel,
  alwaysOpen = false,
  onInputBlur,
  className,
  inputClassName,
  listClassName,
}: ComboboxProps) {
  const base = useId();
  const listId = `${base}-list`;
  const optionId = useCallback((index: number): string => `${base}-option-${index}`, [base]);
  const [popupOpen, setPopupOpen] = useState(false);
  const [active, setActive] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);
  const open = alwaysOpen || (popupOpen && items.length > 0);

  // The list changed under the cursor: start again at the first option.
  // biome-ignore lint/correctness/useExhaustiveDependencies: the reset is keyed on the query and the list size
  useEffect(() => setActive(0), [query, items.length]);

  useEffect(() => {
    listRef.current?.querySelector(`#${CSS.escape(optionId(active))}`)?.scrollIntoView?.({ block: "nearest" });
  }, [active, optionId]);

  function move(delta: number): void {
    if (items.length === 0) return;
    setActive((index) => (index + delta + items.length) % items.length);
  }

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>): void {
    if (event.nativeEvent.isComposing) return;
    switch (event.key) {
      case "ArrowDown":
        event.preventDefault();
        if (!open) setPopupOpen(true);
        else move(1);
        break;
      case "ArrowUp":
        event.preventDefault();
        if (!open) setPopupOpen(true);
        else move(-1);
        break;
      case "Home":
        if (open) {
          event.preventDefault();
          setActive(0);
        }
        break;
      case "End":
        if (open) {
          event.preventDefault();
          setActive(Math.max(items.length - 1, 0));
        }
        break;
      case "Enter": {
        const item = items[active];
        if (open && item) {
          event.preventDefault();
          onSelect(item);
          if (!alwaysOpen) setPopupOpen(false);
        }
        break;
      }
      case "Escape":
        if (!alwaysOpen && open) {
          event.preventDefault();
          event.stopPropagation();
          setPopupOpen(false);
        }
        break;
    }
  }

  const groups: { name: string | undefined; entries: { item: ComboboxItem; index: number }[] }[] = [];
  items.forEach((item, index) => {
    const last = groups[groups.length - 1];
    if (last && last.name === item.group) last.entries.push({ item, index });
    else groups.push({ name: item.group, entries: [{ item, index }] });
  });

  return (
    <div className={className}>
      <input
        type="text"
        role="combobox"
        aria-label={label}
        aria-expanded={open}
        aria-controls={listId}
        aria-autocomplete="list"
        aria-activedescendant={open && items.length > 0 ? optionId(active) : undefined}
        autoComplete="off"
        spellCheck={false}
        placeholder={placeholder}
        value={query}
        onChange={(event) => {
          onQueryChange(event.target.value);
          setPopupOpen(true);
        }}
        onKeyDown={onKeyDown}
        onBlur={() => {
          setPopupOpen(false);
          onInputBlur?.();
        }}
        className={cn(
          "w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring pointer-coarse:min-h-11",
          inputClassName,
        )}
      />
      <div
        id={listId}
        ref={listRef}
        role="listbox"
        aria-label={label}
        hidden={!open}
        className={cn("max-h-80 overflow-y-auto py-1", listClassName)}
      >
        {groups.map((group, groupIndex) => {
          const headingId = `${base}-group-${groupIndex}`;
          const options = group.entries.map(({ item, index }) => (
            // Focus stays in the input, so the option is operated by the pointer only; the keyboard path is the input.
            // biome-ignore lint/a11y/useKeyWithClickEvents: keyboard operation lives on the combobox input
            <div
              key={item.id}
              id={optionId(index)}
              role="option"
              tabIndex={-1}
              aria-selected={index === active}
              onMouseDown={(event) => event.preventDefault()}
              onMouseMove={() => index !== active && setActive(index)}
              onClick={() => {
                onSelect(item);
                if (!alwaysOpen) setPopupOpen(false);
              }}
              className={cn(
                "flex cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-sm pointer-coarse:min-h-11",
                index === active && "bg-accent text-accent-foreground",
              )}
            >
              {item.icon}
              <span className="flex-1">{item.label}</span>
              {item.hint}
            </div>
          ));
          return group.name ? (
            // biome-ignore lint/a11y/useSemanticElements: a group inside a listbox is role=group, not a fieldset
            <div key={group.name} role="group" aria-labelledby={headingId}>
              <div
                id={headingId}
                role="presentation"
                className="px-3 pt-2 pb-1 text-xs font-medium text-muted-foreground"
              >
                {group.name}
              </div>
              {options}
            </div>
          ) : (
            options
          );
        })}
      </div>
      <div role="status" className="sr-only">
        {open ? resultsLabel(items.length) : ""}
      </div>
    </div>
  );
}

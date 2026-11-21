import {
  Book,
  Briefcase,
  Car,
  Coins,
  Film,
  Gift,
  Heart,
  House,
  type LucideIcon,
  PawPrint,
  Plane,
  Shirt,
  ShoppingCart,
  Tag,
  Utensils,
  Wallet,
  Zap,
} from "lucide-react";
import { type Key, useFinanceI18n } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { Field, FieldLabel } from "../ui/field";
import { NativeSelect, NativeSelectOption } from "../ui/native-select";

// The classes are written out in full so Tailwind sees them; the colors are the --vf-cat-* design tokens.
// (An inline style would be blocked by the content security policy on a server-rendered page.)
const swatchClass: Record<string, string> = {
  slate: "bg-[var(--vf-cat-slate)]",
  blue: "bg-[var(--vf-cat-blue)]",
  teal: "bg-[var(--vf-cat-teal)]",
  green: "bg-[var(--vf-cat-green)]",
  olive: "bg-[var(--vf-cat-olive)]",
  amber: "bg-[var(--vf-cat-amber)]",
  orange: "bg-[var(--vf-cat-orange)]",
  red: "bg-[var(--vf-cat-red)]",
  pink: "bg-[var(--vf-cat-pink)]",
  violet: "bg-[var(--vf-cat-violet)]",
};

const inkClass: Record<string, string> = {
  slate: "text-[var(--vf-cat-slate)]",
  blue: "text-[var(--vf-cat-blue)]",
  teal: "text-[var(--vf-cat-teal)]",
  green: "text-[var(--vf-cat-green)]",
  olive: "text-[var(--vf-cat-olive)]",
  amber: "text-[var(--vf-cat-amber)]",
  orange: "text-[var(--vf-cat-orange)]",
  red: "text-[var(--vf-cat-red)]",
  pink: "text-[var(--vf-cat-pink)]",
  violet: "text-[var(--vf-cat-violet)]",
};

const icons: Record<string, LucideIcon> = {
  wallet: Wallet,
  home: House,
  cart: ShoppingCart,
  car: Car,
  heart: Heart,
  film: Film,
  utensils: Utensils,
  briefcase: Briefcase,
  gift: Gift,
  plane: Plane,
  book: Book,
  paw: PawPrint,
  zap: Zap,
  coins: Coins,
  shirt: Shirt,
  tag: Tag,
};

const colorNames: Record<string, Key> = {
  slate: "colorSlate",
  blue: "colorBlue",
  teal: "colorTeal",
  green: "colorGreen",
  olive: "colorOlive",
  amber: "colorAmber",
  orange: "colorOrange",
  red: "colorRed",
  pink: "colorPink",
  violet: "colorViolet",
};

const iconNames: Record<string, Key> = {
  wallet: "iconWallet",
  home: "iconHome",
  cart: "iconCart",
  car: "iconCar",
  heart: "iconHeart",
  film: "iconFilm",
  utensils: "iconUtensils",
  briefcase: "iconBriefcase",
  gift: "iconGift",
  plane: "iconPlane",
  book: "iconBook",
  paw: "iconPaw",
  zap: "iconZap",
  coins: "iconCoins",
  shirt: "iconShirt",
  tag: "iconTag",
};

/** The small badge of a category or a tag: its icon in its color, or a dot when it has no icon. Decorative: the name always sits beside it. */
export function Mark({ color, icon, className }: { color: string; icon?: string; className?: string }) {
  const Icon = icon ? icons[icon] : undefined;
  if (Icon) {
    return (
      <Icon
        aria-hidden="true"
        className={cn("size-4 shrink-0", inkClass[color] ?? "text-muted-foreground", className)}
      />
    );
  }
  return (
    <span
      aria-hidden="true"
      className={cn("size-3 shrink-0 rounded-full border", swatchClass[color] ?? "bg-transparent", className)}
    />
  );
}

/** Radio swatches for the color of a category or tag. Native radios: arrow keys move, and the chosen one carries a ring and a check, not color alone. */
export function ColorPicker({
  name,
  colors,
  value,
  onChange,
}: {
  name: string;
  colors: readonly string[];
  value: string;
  onChange: (color: string) => void;
}) {
  const { t } = useFinanceI18n();
  const options = ["", ...colors];
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="text-sm font-medium">{t("color")}</legend>
      <div className="flex flex-wrap gap-2">
        {options.map((color) => {
          const label = color === "" ? t("noColor") : t(colorNames[color] ?? "noColor");
          return (
            <label
              key={color || "none"}
              className="relative inline-flex size-8 cursor-pointer items-center justify-center rounded-full pointer-coarse:size-11"
            >
              <input
                type="radio"
                name={name}
                value={color}
                checked={value === color}
                onChange={() => onChange(color)}
                className="peer sr-only"
              />
              <span
                aria-hidden="true"
                className={cn(
                  "flex size-6 items-center justify-center rounded-full border border-input text-[0.65rem] font-semibold text-background peer-checked:ring-2 peer-checked:ring-offset-2 peer-checked:ring-foreground peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-ring",
                  color === "" ? "bg-background text-foreground" : swatchClass[color],
                )}
              >
                {value === color ? "✓" : color === "" ? "–" : ""}
              </span>
              <span className="sr-only">{label}</span>
            </label>
          );
        })}
      </div>
    </fieldset>
  );
}

/** A select of the icon list, each with its name. */
export function IconSelect({
  id,
  icons: names,
  value,
  onChange,
}: {
  id: string;
  icons: readonly string[];
  value: string;
  onChange: (icon: string) => void;
}) {
  const { t } = useFinanceI18n();
  return (
    <Field>
      <FieldLabel htmlFor={id}>{t("icon")}</FieldLabel>
      <NativeSelect id={id} value={value} onChange={(event) => onChange(event.target.value)}>
        <NativeSelectOption value="">{t("noIcon")}</NativeSelectOption>
        {names.map((icon) => (
          <NativeSelectOption key={icon} value={icon}>
            {t(iconNames[icon] ?? "noIcon")}
          </NativeSelectOption>
        ))}
      </NativeSelect>
    </Field>
  );
}

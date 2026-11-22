import { HelpHint } from "../../help/HelpHint";
import { type Key, useFinanceI18n } from "../../lib/i18n";
import type { Account } from "../../page-props";
import AmountInput from "../AmountInput";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "../ui/field";
import { Input } from "../ui/input";
import { NativeSelect, NativeSelectOption } from "../ui/native-select";

export interface CardFormData {
  name: string;
  limit: string;
  closingDay: string;
  dueDay: string;
  paymentAccountId: string;
  brand: string;
  last4: string;
}

export const emptyCardForm: CardFormData = {
  name: "",
  limit: "",
  closingDay: "10",
  dueDay: "20",
  paymentAccountId: "",
  brand: "",
  last4: "",
};

const brandKeys: Record<string, Key> = {
  visa: "cardBrandVisa",
  mastercard: "cardBrandMastercard",
  amex: "cardBrandAmex",
  elo: "cardBrandElo",
  hipercard: "cardBrandHipercard",
  diners: "cardBrandDiners",
  other: "cardBrandOther",
};

export interface CardFormProps {
  idPrefix: string;
  data: CardFormData;
  errors: Partial<Record<keyof CardFormData, string>>;
  setData: <K extends keyof CardFormData>(key: K, value: CardFormData[K]) => void;
  accounts: Account[];
  brands: string[];
}

/** The fields of a credit card. Only the last four digits are asked for: never a number, an expiry or a code. */
export default function CardForm({ idPrefix, data, errors, setData, accounts, brands }: CardFormProps) {
  const { t } = useFinanceI18n();
  const id = (name: string): string => `${idPrefix}-${name}`;
  return (
    <FieldGroup className="grid gap-3 sm:grid-cols-2">
      <Field data-invalid={!!errors.name}>
        <FieldLabel htmlFor={id("name")}>{t("cardName")}</FieldLabel>
        <Input
          id={id("name")}
          required
          maxLength={80}
          value={data.name}
          aria-invalid={!!errors.name}
          onChange={(event) => setData("name", event.target.value)}
        />
        <FieldError>{errors.name}</FieldError>
      </Field>
      <Field data-invalid={!!errors.limit}>
        <FieldLabel htmlFor={id("limit")}>{t("cardLimit")}</FieldLabel>
        <AmountInput
          id={id("limit")}
          required
          placeholder="1500.00"
          value={data.limit}
          aria-invalid={!!errors.limit}
          onChange={(event) => setData("limit", event.target.value)}
        />
        <FieldError>{errors.limit}</FieldError>
      </Field>
      <Field data-invalid={!!errors.closingDay}>
        <span className="flex items-center gap-1">
          <FieldLabel htmlFor={id("closing")}>{t("cardClosingDay")}</FieldLabel>
          <HelpHint id="hintCardClosingDay" />
        </span>
        <Input
          id={id("closing")}
          required
          inputMode="numeric"
          maxLength={2}
          value={data.closingDay}
          aria-invalid={!!errors.closingDay}
          onChange={(event) => setData("closingDay", event.target.value)}
        />
        <FieldError>{errors.closingDay}</FieldError>
      </Field>
      <Field data-invalid={!!errors.dueDay}>
        <span className="flex items-center gap-1">
          <FieldLabel htmlFor={id("due")}>{t("cardDueDay")}</FieldLabel>
          <HelpHint id="hintCardDueDay" />
        </span>
        <Input
          id={id("due")}
          required
          inputMode="numeric"
          maxLength={2}
          value={data.dueDay}
          aria-invalid={!!errors.dueDay}
          onChange={(event) => setData("dueDay", event.target.value)}
        />
        <FieldDescription>{t("cardDaysHelp")}</FieldDescription>
        <FieldError>{errors.dueDay}</FieldError>
      </Field>
      <Field data-invalid={!!errors.paymentAccountId}>
        <FieldLabel htmlFor={id("payer")}>{t("cardPaymentAccount")}</FieldLabel>
        <NativeSelect
          id={id("payer")}
          value={data.paymentAccountId}
          onChange={(event) => setData("paymentAccountId", event.target.value)}
        >
          <NativeSelectOption value="">{t("cardNoPaymentAccount")}</NativeSelectOption>
          {accounts.map((account) => (
            <NativeSelectOption key={account.id} value={String(account.id)}>
              {account.name}
            </NativeSelectOption>
          ))}
        </NativeSelect>
        <FieldError>{errors.paymentAccountId}</FieldError>
      </Field>
      <Field data-invalid={!!errors.brand}>
        <FieldLabel htmlFor={id("brand")}>{t("cardBrand")}</FieldLabel>
        <NativeSelect id={id("brand")} value={data.brand} onChange={(event) => setData("brand", event.target.value)}>
          <NativeSelectOption value="">{t("cardNoBrand")}</NativeSelectOption>
          {brands.map((brand) => (
            <NativeSelectOption key={brand} value={brand}>
              {brandKeys[brand] ? t(brandKeys[brand]) : brand}
            </NativeSelectOption>
          ))}
        </NativeSelect>
        <FieldError>{errors.brand}</FieldError>
      </Field>
      <Field data-invalid={!!errors.last4} className="sm:col-span-2">
        <FieldLabel htmlFor={id("last4")}>{t("cardLast4")}</FieldLabel>
        <Input
          id={id("last4")}
          inputMode="numeric"
          autoComplete="off"
          maxLength={4}
          value={data.last4}
          aria-invalid={!!errors.last4}
          onChange={(event) => setData("last4", event.target.value)}
        />
        <FieldDescription>{t("cardLast4Help")}</FieldDescription>
        <FieldError>{errors.last4}</FieldError>
      </Field>
    </FieldGroup>
  );
}

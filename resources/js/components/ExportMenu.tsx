import { Download } from "lucide-react";
import { DropdownMenu } from "radix-ui";
import { useFinanceI18n } from "../lib/i18n";
import { Button } from "./ui/button";

export interface ExportMenuProps {
  csvUrl: string;
  csvExcelUrl: string;
  /** Present for owners and admins only. */
  jsonUrl?: string;
  onPrint?: () => void;
}

const itemClass =
  "flex min-h-8 cursor-default items-center rounded-md px-2.5 py-1.5 text-sm outline-none select-none data-[highlighted]:bg-muted";

/** The Export menu: real links for the downloads (the browser handles the attachments) and a print entry. */
export default function ExportMenu({ csvUrl, csvExcelUrl, jsonUrl, onPrint }: ExportMenuProps) {
  const { t } = useFinanceI18n();
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <Button type="button" variant="outline" data-print="hide">
          <Download aria-hidden="true" data-icon="inline-start" />
          {t("rptExport")}
        </Button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content
          align="end"
          sideOffset={6}
          className="z-50 min-w-52 rounded-lg border bg-popover p-1 text-popover-foreground shadow-md"
        >
          <DropdownMenu.Item asChild className={itemClass}>
            <a href={csvUrl} download>
              {t("rptExportCsv")}
            </a>
          </DropdownMenu.Item>
          <DropdownMenu.Item asChild className={itemClass}>
            <a href={csvExcelUrl} download>
              {t("rptExportCsvExcel")}
            </a>
          </DropdownMenu.Item>
          {jsonUrl && (
            <DropdownMenu.Item asChild className={itemClass}>
              <a href={jsonUrl} download>
                {t("rptExportJson")}
              </a>
            </DropdownMenu.Item>
          )}
          {onPrint && (
            <DropdownMenu.Item className={itemClass} onSelect={onPrint}>
              {t("rptPrint")}
            </DropdownMenu.Item>
          )}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}

import { Download } from "lucide-react";
import { useFinanceI18n } from "../lib/i18n";
import { Button } from "./ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "./ui/dropdown-menu";

export interface ExportMenuProps {
  csvUrl: string;
  csvExcelUrl: string;
  /** Present for owners and admins only. */
  jsonUrl?: string;
  onPrint?: () => void;
}

/** The Export menu: real links for the downloads (the browser handles the attachments) and a print entry. */
export default function ExportMenu({ csvUrl, csvExcelUrl, jsonUrl, onPrint }: ExportMenuProps) {
  const { t } = useFinanceI18n();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant="outline" data-print="hide" shortcutAction="export-transactions">
          <Download aria-hidden="true" data-icon="inline-start" />
          {t("rptExport")}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={6} className="min-w-52">
        <DropdownMenuItem asChild>
          <a href={csvUrl} download>
            {t("rptExportCsv")}
          </a>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <a href={csvExcelUrl} download>
            {t("rptExportCsvExcel")}
          </a>
        </DropdownMenuItem>
        {jsonUrl && (
          <DropdownMenuItem asChild>
            <a href={jsonUrl} download>
              {t("rptExportJson")}
            </a>
          </DropdownMenuItem>
        )}
        {onPrint && <DropdownMenuItem onSelect={onPrint}>{t("rptPrint")}</DropdownMenuItem>}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

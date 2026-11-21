import { Text } from "@react-email/components";

import { go } from "./go";
import { Shell } from "./Layout";

// Props mirror the Go mailable's fields (app/mail/finance.go). The defaults are
// Go template actions: `email export` writes them into the HTML and Go escapes
// the real values when the mail is sent.
type Props = { workspace: string; amount: string; months: string; validUntil: string; url: string };

export default function PixReceipt({
  workspace = go(".Workspace"),
  amount = go(".Amount"),
  months = go(".Months"),
  validUntil = go(".ValidUntil"),
  url = go(".URL"),
}: Partial<Props>) {
  return (
    <Shell preview="We received your PIX payment" title="PIX payment received" action="Open billing" href={url}>
      <Text>
        We received {amount} for {months} month(s) of Pro for {workspace}. Pro lasts until {validUntil}. A PIX payment
        does not renew by itself: pay again before that date to keep Pro.
      </Text>
    </Shell>
  );
}

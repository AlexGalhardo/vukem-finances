import { Text } from "@react-email/components";

import { go } from "./go";
import { Shell } from "./Layout";

// Props mirror the Go mailable's fields (app/mail/finance.go). The defaults are
// Go template actions: `email export` writes them into the HTML and Go escapes
// the real values when the mail is sent.
type Props = { workspace: string; title: string; body: string; url: string };

export default function AlertNotice({
  workspace = go(".Workspace"),
  title = go(".Title"),
  body = go(".Body"),
  url = go(".URL"),
}: Partial<Props>) {
  return (
    <Shell preview={title} title={title} action={`Open ${workspace}`} href={url}>
      <Text>{body}</Text>
    </Shell>
  );
}

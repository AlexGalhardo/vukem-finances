import { Text } from "@react-email/components";

import { go } from "./go";
import { Shell } from "./Layout";

// Props mirror the Go mailable's fields (app/mail/finance.go). The defaults are
// Go template actions: `email export` writes them into the HTML and Go escapes
// the real values when the mail is sent.
type Props = { workspace: string; reason: string; url: string };

export default function AccessRevoked({
  workspace = go(".Workspace"),
  reason = go(".Reason"),
  url = go(".URL"),
}: Partial<Props>) {
  return (
    <Shell preview="Your Pro access has ended" title="Your Pro access has ended" action="Open billing" href={url}>
      <Text>
        Pro for {workspace} has ended before the end of the paid period because {reason}. The workspace is on the Free
        plan now. You can subscribe again from the billing page.
      </Text>
    </Shell>
  );
}

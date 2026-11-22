import { Text } from "@react-email/components";

import { go } from "./go";
import { Shell } from "./Layout";

// Props mirror the Go mailable's fields (app/mail/finance.go). The defaults are
// Go template actions: `email export` writes them into the HTML and Go escapes
// the real values when the mail is sent.
type Props = { workspace: string; card: string; remaining: string; dueOn: string; url: string };

export default function StatementDue({
  workspace = go(".Workspace"),
  card = go(".Card"),
  remaining = go(".Remaining"),
  dueOn = go(".DueOn"),
  url = go(".URL"),
}: Partial<Props>) {
  return (
    <Shell
      preview="A card statement is about to fall due"
      title={`${card} statement is due ${dueOn}`}
      action="See the statement"
      href={url}
    >
      <Text>
        {workspace} still owes {remaining} on the {card} statement, due on {dueOn}.
      </Text>
    </Shell>
  );
}

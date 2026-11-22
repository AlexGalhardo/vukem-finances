import { Text } from "@react-email/components";

import { go } from "./go";
import { Shell } from "./Layout";

// Props mirror the Go mailable's fields (app/mail/finance.go). The defaults are
// Go template actions: `email export` writes them into the HTML and Go escapes
// the real values when the mail is sent.
type Props = { workspace: string; goal: string; target: string; url: string };

export default function GoalReached({
  workspace = go(".Workspace"),
  goal = go(".Goal"),
  target = go(".Target"),
  url = go(".URL"),
}: Partial<Props>) {
  return (
    <Shell preview="A savings goal was reached" title={`${goal} reached its target`} action="See your goals" href={url}>
      <Text>
        {workspace} has saved the {target} target of {goal}.
      </Text>
    </Shell>
  );
}

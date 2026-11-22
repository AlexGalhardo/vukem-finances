import { Text } from "@react-email/components";

import { go } from "./go";
import { Shell } from "./Layout";

// Props mirror the Go mailable's fields (app/mail/privacy.go).
type Props = { workspace: string; url: string };

export default function WorkspaceDeletionCancelled({ workspace = go(".Workspace"), url = go(".URL") }: Partial<Props>) {
  return (
    <Shell
      preview="The deletion of a workspace was cancelled"
      title={`${workspace} will not be deleted`}
      action="Open the workspace"
      href={url}
    >
      <Text>The owner cancelled the deletion of {workspace}. It is open for changes again.</Text>
    </Shell>
  );
}

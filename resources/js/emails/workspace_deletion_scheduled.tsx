import { Text } from "@react-email/components";

import { go } from "./go";
import { Shell } from "./Layout";

// Props mirror the Go mailable's fields (app/mail/privacy.go).
type Props = { workspace: string; purgeDate: string; url: string };

export default function WorkspaceDeletionScheduled({
  workspace = go(".Workspace"),
  purgeDate = go(".PurgeDate"),
  url = go(".URL"),
}: Partial<Props>) {
  return (
    <Shell
      preview="A workspace you belong to will be deleted"
      title={`${workspace} will be deleted`}
      action="Open the workspace"
      href={url}
    >
      <Text>
        The owner asked to delete {workspace}. It is read-only from now on, and everything in it is erased for good on{" "}
        {purgeDate}. You can still read and export your data until then. Only the owner can cancel the deletion.
      </Text>
    </Shell>
  );
}

import { Text } from "@react-email/components";

import { go } from "./go";
import { Shell } from "./Layout";

// Props mirror the Go mailable's fields (app/mail/privacy.go).
type Props = { name: string; purgeDate: string; url: string };

export default function AccountDeletionScheduled({
  name = go(".Name"),
  purgeDate = go(".PurgeDate"),
  url = go(".URL"),
}: Partial<Props>) {
  return (
    <Shell
      preview="Your account will be deleted"
      title="Your account will be deleted"
      action="Cancel the deletion"
      href={url}
    >
      <Text>
        Hi {name}, you asked to delete your account. Every session and access token was ended, and the account is
        erased for good on {purgeDate}. To keep it, sign in before then and cancel the deletion. If this was not you,
        sign in and cancel it now, then change your password.
      </Text>
    </Shell>
  );
}

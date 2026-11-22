const dict = {
  privacy: "Privacy and data",
  privacyIntro: "Take your data with you, or erase it. Each action asks for your password first.",
  privacyPassword: "Your password",
  privacyPasswordHint: "Signed up with Google and never chose a password? Reset it from the sign-in page first.",
  privacyExportTitle: "Export my data",
  privacyExportBody:
    "A JSON file with your profile, your memberships, the list of your sign-in sessions and tokens (never their values), and everything in the workspaces you own. Passwords, two-factor secrets, tokens and AI keys are never included.",
  privacyExportButton: "Export my data",
  privacyExportStarted: "Your download started. If it did not, confirm your password again.",
  privacyWorkspaceTitle: "Delete this workspace",
  privacyWorkspaceBody:
    "Only the owner can do this. The workspace becomes read-only for everyone now, its members are told by email, and everything in it is erased for good after {days} days. The owner can cancel until then. The subscription is cancelled at once; there is no automatic refund.",
  privacyWorkspaceName: "Type the workspace name to confirm",
  privacyWorkspaceDelete: "Delete the workspace",
  privacyWorkspaceOwnerOnly: "Only the owner of this workspace can delete it.",
  privacyWorkspacePersonal:
    "The personal edition has one workspace. To remove it and everything in it, erase all local data below.",
  privacyPendingTitle: "This workspace is scheduled for deletion",
  privacyPendingBody:
    "It is read-only and will be erased for good on {date}. You can still read and export your data until then.",
  privacyPendingCancel: "Cancel the deletion",
  privacyPendingOwnerOnly: "Only the owner can cancel the deletion.",
  privacyAccountTitle: "Delete my account",
  privacyAccountBody:
    "Every session and token ends now and the account is erased for good after {days} days; sign in before then to cancel. Workspaces where you are the only member go with it. If you are the only owner of a workspace other people use, transfer the ownership or delete that workspace first. In the workspaces that remain, your private data (assistant conversations, notifications, tokens, devices) is erased and your name is removed from what you wrote.",
  privacyAccountConfirm: "Type your email address to confirm",
  privacyAccountDelete: "Delete my account",
  accountDeletionTitle: "Your account is scheduled for deletion",
  accountDeletionBody:
    "It will be erased for good on {date}. Signing in does not cancel it. Your sessions, tokens and connected apps were ended and are not restored.",
  accountDeletionCancel: "Keep my account",
  accountDeletionSignOut: "Sign out",
  privacyEraseTitle: "Erase all local data",
  privacyEraseBody:
    "Removes the database of this installation, with everything in it, from this device. Nothing is sent anywhere and no network is needed. The app then restarts empty, at the first setup. Export your data first if you want to keep it.",
  privacyEraseConfirm: "Type ERASE to confirm",
  privacyEraseButton: "Erase all local data",
  privacyErased: "The local data was erased. The app is restarting empty: reload this page in a few seconds.",
} as const;
export default dict;

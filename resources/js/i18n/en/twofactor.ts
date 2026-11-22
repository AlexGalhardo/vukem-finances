const dict = {
  tfaTitle: "Two-factor authentication",
  tfaStatusTitle: "Status",
  tfaOn: "Two-factor authentication is on: logging in asks for a code from your authenticator app.",
  tfaOff: "Add a second step to your login: a code from an authenticator app on your phone.",
  tfaEnable: "Enable two-factor authentication",
  tfaSetupTitle: "Set up",
  tfaScan:
    "Scan this QR code with your authenticator app, or enter the setup key by hand, then type the code it shows.",
  tfaQrAlt: "Two-factor setup QR code",
  tfaSetupKey: "Setup key",
  tfaCode: "Code",
  tfaConfirm: "Confirm",
  tfaRecoveryTitle: "Recovery codes",
  tfaRecoveryDescription: "Single-use codes for the day you lose your phone.",
  tfaHelpRecovery: "About recovery codes",
  tfaRecoveryStore:
    "Store these recovery codes somewhere safe. Each one works once if you lose your device; they are not shown again.",
  tfaCopyCodes: "Copy codes",
  tfaCopied: "Copied",
  tfaRecoveryLeft: "You have {count} recovery codes left.",
  tfaCurrentPassword: "Current password",
  tfaRegenerate: "Regenerate recovery codes",
  tfaTurnOff: "Turn off",
  tfaTurnOffDescription: "Login goes back to the password alone. You can turn it on again at any time.",
  tfaDisable: "Disable two-factor authentication",
  tfaBack: "Back to the dashboard",
} as const;
export default dict;

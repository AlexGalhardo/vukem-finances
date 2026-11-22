import { usePage } from "@inertiajs/react";
import type { ReactNode } from "react";
import ReleaseBadge from "../components/ReleaseBadge";
import { Alert, AlertDescription } from "../components/ui/alert";
import { TooltipProvider } from "../components/ui/tooltip";
import type { SharedProps } from "../page-props";
import AuthLayout from "./AuthLayout";
import Shell from "./Shell";

/** Every screen, whatever its frame, shows the running release. */
export default function AppLayout({ children }: { children: ReactNode }) {
  return (
    <>
      <Frame>{children}</Frame>
      <ReleaseBadge />
    </>
  );
}

/** Chooses the frame: the panel's own layout, the signed-in Shell, or the signed-out AuthLayout. */
function Frame({ children }: { children: ReactNode }) {
  const page = usePage<SharedProps>();
  const { auth, workspace } = page.props;
  const success = page.flash?.success ?? page.flash?.status;
  const error = page.flash?.error;
  const notices = (
    <>
      {typeof success === "string" && (
        <Alert variant="success">
          <AlertDescription>{success}</AlertDescription>
        </Alert>
      )}
      {typeof error === "string" && (
        <Alert variant="error">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
    </>
  );

  // The panel's own screens, Revenue included, bring AdminLayout (sidebar, title, theme): wrapping them here would
  // stack a second header.
  if (page.component.startsWith("Admin/")) return <>{children}</>;

  if (auth?.user && workspace) return <Shell notices={notices}>{children}</Shell>;

  return (
    <TooltipProvider>
      <AuthLayout notices={notices}>{children}</AuthLayout>
    </TooltipProvider>
  );
}

import { createContext, useContext } from "react";

export interface HelpActions {
  /** Opens the page tutorial at a step; set by the tutorial host, absent until it mounts. */
  openTutorialStep?: (screenId: string, step: string) => void;
}

export const HelpActionsContext = createContext<HelpActions>({});

export function useHelpActions(): HelpActions {
  return useContext(HelpActionsContext);
}

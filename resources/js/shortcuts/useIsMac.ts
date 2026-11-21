import { useEffect, useState } from "react";
import { detectMac } from "./dispatcher";

/** False on the server and on first render, so markup matches before hydration; true on macOS afterwards. */
export function useIsMac(): boolean {
  const [mac, setMac] = useState(false);
  useEffect(() => setMac(detectMac()), []);
  return mac;
}

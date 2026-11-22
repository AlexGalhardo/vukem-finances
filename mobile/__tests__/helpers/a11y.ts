import type { ReactTestRendererJSON } from "react-test-renderer";

type Node = ReactTestRendererJSON;
type Json = Node | Node[] | null;

export const MIN_TARGET = 48;

function nodes(json: Json): Node[] {
  if (json === null) return [];
  return Array.isArray(json) ? json : [json];
}

function children(node: Node): Node[] {
  return (node.children ?? []).filter((c): c is Node => typeof c !== "string");
}

function flatten(style: unknown): Record<string, unknown> {
  if (Array.isArray(style)) return Object.assign({}, ...style.map(flatten));
  return style !== null && typeof style === "object" ? (style as Record<string, unknown>) : {};
}

function text(node: Node): string {
  return (node.children ?? []).map((c) => (typeof c === "string" ? c : text(c))).join("");
}

function hasText(node: Node): boolean {
  return children(node).some((c) => c.type === "Text" || hasText(c)) || node.type === "Text";
}

/** A host view the person can press: React Native's Pressable renders these responder props. */
function isTouchable(node: Node): boolean {
  const p = node.props;
  return typeof p.onClick === "function" || typeof p.onResponderRelease === "function";
}

function minHeightOf(node: Node): number {
  const own = Number(flatten(node.props.style).minHeight ?? 0);
  return Math.max(own, ...children(node).map((c) => Number(flatten(c.props.style).minHeight ?? 0)));
}

function walk(node: Node, visit: (n: Node) => void): void {
  visit(node);
  for (const c of children(node)) walk(c, visit);
}

/** Layout rules that keep text readable at 200 percent font size: nothing that holds text has a fixed size, clips or truncates. */
export function layoutViolations(json: Json): string[] {
  const out: string[] = [];
  for (const root of nodes(json)) {
    walk(root, (n) => {
      const style = flatten(n.props.style);
      if (n.type === "Text") {
        if (n.props.numberOfLines !== undefined) out.push(`Text "${text(n)}" sets numberOfLines`);
        if (n.props.allowFontScaling === false) out.push(`Text "${text(n)}" turns font scaling off`);
        const cap = n.props.maxFontSizeMultiplier;
        if (typeof cap === "number" && cap < 2) out.push(`Text "${text(n)}" caps font scale at ${cap}`);
        for (const k of ["height", "maxHeight"]) {
          if (style[k] !== undefined) out.push(`Text "${text(n)}" has a fixed ${k}`);
        }
        return;
      }
      if (!hasText(n)) return;
      for (const k of ["height", "maxHeight", "width", "maxWidth"]) {
        const v = style[k];
        if (typeof v === "number") out.push(`${n.type} holding text has a fixed ${k} of ${v}`);
      }
      if (style.overflow === "hidden") out.push(`${n.type} holding text clips with overflow hidden`);
    });
  }
  return out;
}

/** Every touchable has a role, a label and a target of at least 48 dp; every input has a label; every radio and checkbox says if it is on. */
export function a11yViolations(json: Json): string[] {
  const out: string[] = [];
  for (const root of nodes(json)) {
    walk(root, (n) => {
      const p = n.props;
      if (n.type === "TextInput") {
        if (typeof p.accessibilityLabel !== "string" || p.accessibilityLabel === "")
          out.push("a TextInput has no accessibilityLabel");
        if (Number(flatten(p.style).minHeight ?? 0) < MIN_TARGET)
          out.push(`TextInput "${String(p.accessibilityLabel)}" is under ${MIN_TARGET} dp tall`);
        return;
      }
      if (!isTouchable(n)) return;
      const name =
        typeof p.accessibilityLabel === "string" && p.accessibilityLabel !== "" ? p.accessibilityLabel : text(n);
      if (typeof p.accessibilityRole !== "string") out.push(`a touchable "${name}" has no accessibilityRole`);
      if (name.trim() === "") out.push("a touchable has no accessibilityLabel and no text");
      if (minHeightOf(n) < MIN_TARGET) out.push(`touchable "${name}" is under ${MIN_TARGET} dp tall`);
      if (
        (p.accessibilityRole === "radio" || p.accessibilityRole === "checkbox") &&
        p.accessibilityState?.checked === undefined
      ) {
        out.push(`${String(p.accessibilityRole)} "${name}" does not expose its checked state`);
      }
    });
  }
  return out;
}

export function hasHeading(json: Json): boolean {
  let found = false;
  for (const root of nodes(json)) walk(root, (n) => (found ||= n.props.accessibilityRole === "header"));
  return found;
}

export function touchableCount(json: Json): number {
  let n = 0;
  for (const root of nodes(json)) walk(root, (node) => (n += isTouchable(node) ? 1 : 0));
  return n;
}

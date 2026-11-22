import { evaluate } from "@mdx-js/mdx";
import { createElement } from "react";
import * as runtime from "react/jsx-runtime";
import { renderToStaticMarkup } from "react-dom/server";
import rehypeSlug from "rehype-slug";
import remarkGfm from "remark-gfm";
import { bundledLanguages, codeToHast } from "shiki";

export interface Heading {
  level: 2 | 3 | 4;
  id: string;
  text: string;
}

export interface Rendered {
  html: string;
  headings: Heading[];
}

interface HNode {
  type: string;
  tagName?: string;
  value?: string;
  properties?: Record<string, unknown>;
  children?: HNode[];
}

function textOf(node: HNode): string {
  if (node.type === "text") return node.value ?? "";
  return (node.children ?? []).map(textOf).join("");
}

const ALIASES: Record<string, string> = { sh: "bash", shell: "bash", console: "bash", env: "ini", dotenv: "ini" };

function languageOf(code: HNode): string {
  const classes = (code.properties?.className as string[] | undefined) ?? [];
  const lang = classes.find((c) => c.startsWith("language-"))?.slice("language-".length) ?? "text";
  const name = ALIASES[lang] ?? lang;
  return name === "text" || name in bundledLanguages ? name : "text";
}

/** Highlights fenced code with Shiki, as inline CSS variables for both themes so no script is needed. */
function highlight(): (tree: HNode) => Promise<void> {
  return async (tree) => {
    const jobs: Promise<void>[] = [];
    const visit = (node: HNode): void => {
      for (const child of node.children ?? []) {
        const code = child.tagName === "pre" ? child.children?.find((c) => c.tagName === "code") : undefined;
        if (child.tagName === "pre" && code) {
          const lang = languageOf(code);
          const source = textOf(code).replace(/\n$/, "");
          jobs.push(
            codeToHast(source, {
              lang,
              themes: { light: "github-light", dark: "github-dark" },
              defaultColor: false,
            }).then((root) => {
              const pre = (root.children[0] as unknown as HNode) ?? child;
              pre.properties = {
                ...pre.properties,
                tabIndex: 0,
                role: "region",
                ariaLabel: lang === "text" ? "Code block" : `Code block, ${lang}`,
                className: ["code"],
              };
              Object.assign(child, pre);
            }),
          );
        } else visit(child);
      }
    };
    visit(tree);
    await Promise.all(jobs);
  };
}

/** Wraps every table in a focusable, named region, so a wide table scrolls by itself. */
function scrollTables(): (tree: HNode) => void {
  return (tree) => {
    let heading = "page";
    const visit = (node: HNode): void => {
      const children = node.children ?? [];
      children.forEach((child, i) => {
        if (/^h[1-6]$/.test(child.tagName ?? "")) heading = textOf(child);
        if (child.tagName === "table") {
          children[i] = {
            type: "element",
            tagName: "div",
            properties: { className: ["table-scroll"], tabIndex: 0, role: "region", ariaLabel: `Table: ${heading}` },
            children: [child],
          };
        } else visit(child);
      });
    };
    visit(tree);
  };
}

/** Puts the base path in front of every root-relative link. */
function basePathLinks(base: string): () => (tree: HNode) => void {
  return () => (tree) => {
    const visit = (node: HNode): void => {
      if (node.tagName === "a" && typeof node.properties?.href === "string") {
        const href = node.properties.href;
        if (href.startsWith("/") && !href.startsWith("//")) node.properties.href = `${base}${href}`;
      }
      node.children?.forEach(visit);
    };
    visit(tree);
  };
}

function collectHeadings(out: Heading[]): () => (tree: HNode) => void {
  return () => (tree) => {
    const visit = (node: HNode): void => {
      const match = node.tagName?.match(/^h([2-4])$/);
      if (match && typeof node.properties?.id === "string") {
        out.push({ level: Number(match[1]) as 2 | 3 | 4, id: node.properties.id, text: textOf(node) });
      }
      node.children?.forEach(visit);
    };
    visit(tree);
  };
}

/** Renders the Markdown of a page to HTML. Plain Markdown with GFM: no JSX, no script, no raw HTML. */
export async function renderMarkdown(body: string, base = ""): Promise<Rendered> {
  const headings: Heading[] = [];
  const mod = await evaluate(body, {
    ...(runtime as object),
    format: "md",
    remarkPlugins: [remarkGfm],
    rehypePlugins: [rehypeSlug, collectHeadings(headings), highlight, scrollTables, basePathLinks(base)],
  } as unknown as Parameters<typeof evaluate>[1]);
  const html = renderToStaticMarkup(createElement(mod.default as () => ReturnType<typeof createElement>));
  return { html, headings };
}

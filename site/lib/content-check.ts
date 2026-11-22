import { type NavSection, navigation } from "../content/navigation.ts";
import { buildNavigation, type DocPage, type NavGroup, orderedSlugs } from "./docs.ts";
import { renderMarkdown } from "./render.ts";

/** The longest title: the page title is followed by " | vukem-finances" and the whole stays at 60 characters. */
export const maxTitle = 60 - " | vukem-finances".length;
export const maxDescription = 160;
export const bannerStart = "> Generated from `";

/** Problems of the front matter, the Markdown and the navigation of the pages. */
export async function checkContent(pages: DocPage[], sections: NavSection[] = navigation): Promise<string[]> {
  const problems: string[] = [];
  const fail = (page: DocPage, line: number, message: string): number =>
    problems.push(`${page.file}:${line}: ${message}`);

  for (const page of pages) {
    if (!page.title.trim()) fail(page, 1, "front matter needs a title");
    else if ([...page.title].length > maxTitle)
      fail(page, 1, `title is ${[...page.title].length} characters, at most ${maxTitle}`);
    if (!page.description.trim()) fail(page, 1, "front matter needs a description");
    else if ([...page.description].length > maxDescription)
      fail(page, 1, `description is over ${maxDescription} characters`);
    if (page.generated && !page.body.trimStart().startsWith(bannerStart))
      fail(page, page.bodyLine, "a generated page must start with the generated banner");

    let fence = false;
    page.body.split("\n").forEach((raw, i) => {
      const line = page.bodyLine + i;
      const open = raw.match(/^\s*(```+|~~~+)\s*(\S*)/);
      if (open) {
        if (!fence && !open[2]) fail(page, line, "a fenced code block needs a language");
        fence = !fence;
        return;
      }
      if (fence) return;
      if (/^# /.test(raw)) fail(page, line, "the page title is the only h1: start headings at h2");
      if (/<\/?[A-Za-z!]/.test(raw.replace(/`[^`]*`/g, ""))) fail(page, line, "raw HTML is not allowed: use Markdown");
    });

    try {
      const { headings } = await renderMarkdown(page.body);
      let previous = 1;
      for (const h of headings) {
        if (h.level > previous + 1)
          fail(page, page.bodyLine, `heading "${h.text}" jumps from h${previous} to h${h.level}`);
        previous = h.level;
      }
    } catch (error) {
      fail(page, page.bodyLine, `does not compile: ${(error as Error).message}`);
    }
  }

  let groups: NavGroup[];
  try {
    groups = buildNavigation(pages, sections);
  } catch (error) {
    problems.push(`content/navigation.ts: ${(error as Error).message}`);
    return problems;
  }
  const listed = orderedSlugs(groups);
  for (const slug of new Set(listed)) {
    if (listed.filter((s) => s === slug).length > 1) problems.push(`content/navigation.ts: "${slug}" is listed twice`);
  }
  for (const page of pages) {
    if (!listed.includes(page.slug)) problems.push(`${page.file}: the page is not in content/navigation.ts`);
  }
  return problems;
}

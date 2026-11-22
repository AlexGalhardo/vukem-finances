import matter from "gray-matter";

const refuse = (): never => {
  throw new Error("JavaScript front matter is not allowed");
};

// gray-matter 4.0.3 evaluates a block that opens with ---js or ---javascript, whatever the options say, so both
// the opening line and the engines are locked down. Passing options also keeps matter() from its input cache.
export const frontMatterOptions = { engines: { javascript: { parse: refuse, stringify: refuse } } };

export interface FrontMatter {
  data: Record<string, unknown>;
  content: string;
}

/** Parses the YAML front matter of a page. `file` only names the page in an error. */
export function parseFrontMatter(raw: string, file: string): FrontMatter {
  const firstLine = raw.split("\n", 1)[0].replace(/\r$/, "");
  if (firstLine !== "---") throw new Error(`${file}: a page must open with a line that is exactly "---"`);
  const parsed = matter(raw, frontMatterOptions);
  return { data: parsed.data as Record<string, unknown>, content: parsed.content };
}

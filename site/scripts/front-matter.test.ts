import { afterAll, describe, expect, test } from "bun:test";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import matter from "gray-matter";
import { readPage } from "../lib/docs.ts";
import { frontMatterOptions, parseFrontMatter } from "../lib/front-matter.ts";

const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "vf-matter-"));
afterAll(() => fs.rmSync(tmp, { recursive: true, force: true }));

const marker = "__vfFrontMatterRan";
const g = globalThis as Record<string, unknown>;

function page(name: string, source: string): string {
  const file = path.join(tmp, name);
  fs.writeFileSync(file, source);
  return file;
}

describe("front matter never runs code", () => {
  test("a page that opens with ---js is refused, naming the file, and its code never runs", () => {
    delete g[marker];
    const file = page("js.md", `---js\n{ title: (globalThis.${marker} = true, "x") }\n---\nBody\n`);
    expect(() => readPage(tmp, file, false)).toThrow(/js\.md: a page must open with a line that is exactly "---"/);
    expect(g[marker]).toBeUndefined();
  });

  test("---javascript and a padded or prefixed opening line are refused too", () => {
    delete g[marker];
    for (const [i, opening] of ["---javascript", "--- ", " ---", "﻿---", "---yaml"].entries()) {
      const file = page(`bad${i}.md`, `${opening}\n{ title: (globalThis.${marker} = true, "x") }\n---\nBody\n`);
      expect(() => readPage(tmp, file, false), opening).toThrow(/exactly "---"/);
    }
    expect(g[marker]).toBeUndefined();
  });

  test("the engine itself refuses JavaScript, through both names, and never runs it", () => {
    delete g[marker];
    const body = `{ title: (globalThis.${marker} = true, "x") }`;
    expect(() => matter(`---js\n${body}\n---\nBody\n`, frontMatterOptions)).toThrow(
      "JavaScript front matter is not allowed",
    );
    expect(() => matter(`---javascript\n${body}\n---\nBody\n`, frontMatterOptions)).toThrow(
      "JavaScript front matter is not allowed",
    );
    expect(g[marker]).toBeUndefined();
  });

  test("options are passed, so a parse never reuses a cached result of the default engines", () => {
    const source = `---js\n{ title: "x" }\n---\nBody\n`;
    matter.cache = {};
    expect(() => matter(source, frontMatterOptions)).toThrow();
    expect(Object.keys(matter.cache)).toEqual([]);
  });

  test("a normal YAML page still parses", () => {
    const file = page("ok.md", '---\ntitle: "Ok"\ndescription: "Fine."\n---\n\n## Body\n');
    expect(parseFrontMatter(fs.readFileSync(file, "utf8"), "ok.md").data).toEqual({
      title: "Ok",
      description: "Fine.",
    });
    expect(readPage(tmp, file, false)).toMatchObject({ slug: "ok", title: "Ok", description: "Fine." });
  });
});

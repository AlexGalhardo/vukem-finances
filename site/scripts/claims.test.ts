import { describe, expect, test } from "bun:test";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { catalogs, flatten } from "../lib/copy.ts";
import { checkCatalogs, checkRegister, findForbidden } from "../lib/claims.ts";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..");
const brief = await Bun.file(path.join(repoRoot, "docs", "landing-brief.md")).text();

describe("claims register", () => {
  test("the committed register and catalogs pass", () => {
    expect(checkRegister(brief, repoRoot)).toEqual([]);
    expect(checkCatalogs()).toEqual([]);
  });

  test("a row pointing at a missing file fails", () => {
    const broken = brief.replace("`LICENSE`", "`does/not/exist.md`");
    expect(checkRegister(broken, repoRoot).join("\n")).toContain("missing evidence does/not/exist.md");
  });

  test("a row without evidence fails", () => {
    const broken = brief.replace("`LICENSE`", "none");
    expect(checkRegister(broken, repoRoot).join("\n")).toContain("no evidence path");
  });

  test("a where key that is not in the catalog fails", () => {
    const broken = brief.replace("hero.eyebrow,", "hero.nothing,");
    expect(checkRegister(broken, repoRoot).join("\n")).toContain("unknown catalog key hero.nothing");
  });
});

describe("forbidden phrases", () => {
  test("each language is caught", () => {
    expect(findForbidden("Start free today")).toContain("start free");
    expect(findForbidden("Comece grátis agora")).toContain("comece grátis");
    expect(findForbidden("Empieza gratis ya")).toContain("empieza gratis");
  });

  test("word boundaries: a buyer is not buy", () => {
    expect(findForbidden("a buyer and a login-free page")).toEqual(["login"]);
    expect(findForbidden("the buyer")).toEqual([]);
  });

  test("a catalog with Start free fails the scan", () => {
    const copy = structuredClone(catalogs.en);
    copy.hero.ctaDownload = "Start free";
    const hits = Object.values(flatten(copy)).flatMap((v) => findForbidden(v, "en"));
    expect(hits).toEqual(["start free"]);
  });
});

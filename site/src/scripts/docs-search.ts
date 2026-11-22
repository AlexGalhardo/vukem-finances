// Progressive enhancement of the documentation search. The container is hidden in the HTML and shown here, so a
// browser without JavaScript never sees a control that cannot work. The only requests are same-origin GETs of the
// static Pagefind files under <base>/pagefind/ (D-020), made by Pagefind itself.
interface PagefindData {
  url: string;
  excerpt: string;
  meta: { title?: string };
}

interface Pagefind {
  search(query: string): Promise<{ results: { data(): Promise<PagefindData> }[] }>;
}

const MAX_RESULTS = 8;

const root = document.querySelector<HTMLElement>("[data-docs-search]");
const input = root?.querySelector<HTMLInputElement>("input");
const live = root?.querySelector<HTMLElement>("[data-search-status]");
const list = root?.querySelector<HTMLUListElement>("[data-search-results]");

if (root && input && live && list) {
  const base = root.dataset.base ?? "";
  let engine: Promise<Pagefind> | undefined;
  let latest = 0;

  const load = (): Promise<Pagefind> => {
    engine ??= import(/* @vite-ignore */ `${base}/pagefind/pagefind.js`) as Promise<Pagefind>;
    return engine;
  };

  // The excerpt wraps matches in <mark>; text content keeps only the words.
  const plain = (html: string): string => new DOMParser().parseFromString(html, "text/html").body.textContent ?? "";

  const render = (query: string, items: PagefindData[]): void => {
    list.replaceChildren(
      ...items.map((item) => {
        const li = document.createElement("li");
        const a = document.createElement("a");
        a.href = `${base}${item.url}`;
        a.textContent = item.meta.title ?? item.url;
        const p = document.createElement("p");
        p.textContent = plain(item.excerpt);
        li.append(a, p);
        return li;
      }),
    );
    live.textContent = items.length === 0 ? `No results for "${query}".` : `${items.length} results for "${query}".`;
  };

  const run = async (): Promise<void> => {
    const query = input.value.trim();
    const mine = ++latest;
    if (query === "") {
      list.replaceChildren();
      live.textContent = "";
      return;
    }
    try {
      const found = await (await load()).search(query);
      const items = await Promise.all(found.results.slice(0, MAX_RESULTS).map((r) => r.data()));
      if (mine === latest) render(query, items);
    } catch {
      engine = undefined;
      if (mine === latest) live.textContent = "Search is not available in this build.";
    }
  };

  input.addEventListener("input", () => void run());
  input.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && input.value !== "") {
      input.value = "";
      void run();
    }
  });
  root.hidden = false;
}

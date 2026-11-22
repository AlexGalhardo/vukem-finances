// A small renderer for the legal drafts in docs/legal: headings, paragraphs, dash lists, links, bold and code.
// Everything is escaped first, so a draft cannot inject markup.
function escapeHtml(text: string): string {
  return text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

function inline(text: string): string {
  return escapeHtml(text)
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/\[([^\]]+)\]\((https?:\/\/[^)\s]+|\/[^)\s]*)\)/g, '<a href="$2">$1</a>');
}

/** Renders the markdown to HTML and returns the text of the first heading as `title`. */
export function renderMarkdown(source: string): { html: string; title: string } {
  const out: string[] = [];
  let title = "";
  let paragraph: string[] = [];
  let list: string[] = [];

  const flush = (): void => {
    if (paragraph.length > 0) out.push(`<p>${inline(paragraph.join(" "))}</p>`);
    if (list.length > 0) out.push(`<ul>${list.map((item) => `<li>${inline(item)}</li>`).join("")}</ul>`);
    paragraph = [];
    list = [];
  };

  for (const line of source.replace(/\r\n/g, "\n").split("\n")) {
    const heading = line.match(/^(#{1,3})\s+(.*)$/);
    if (heading) {
      flush();
      const level = heading[1].length;
      if (level === 1 && !title) title = heading[2];
      out.push(`<h${level}>${inline(heading[2])}</h${level}>`);
    } else if (/^-\s+/.test(line)) {
      if (paragraph.length > 0) flush();
      list.push(line.replace(/^-\s+/, ""));
    } else if (line.trim() === "") {
      flush();
    } else {
      if (list.length > 0) flush();
      paragraph.push(line.trim());
    }
  }
  flush();
  return { html: out.join("\n"), title };
}

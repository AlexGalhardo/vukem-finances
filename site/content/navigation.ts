// The sidebar of the documentation. An item is the slug of a page (its path under content/docs or
// content/generated, without the extension) or a directory whose pages are listed in order.
export type NavItem = string | { dir: string };
export interface NavSection {
  title: string;
  items: NavItem[];
}

export const navigation: NavSection[] = [
  {
    "title": "Start",
    "items": [
      "index",
      "start/what-is-it",
      "start/editions",
      "start/quick-start"
    ]
  },
  {
    "title": "Run it",
    "items": [
      "run/docker",
      "run/vps",
      "run/desktop",
      "run/mobile",
      "run/offline",
      "run/upgrading",
      "run/backup-restore"
    ]
  },
  {
    "title": "Use it",
    "items": [
      "use/accounts-transfers",
      "use/transactions-import",
      "use/cards",
      "use/recurrences-installments",
      "use/budgets-goals",
      "use/insights-exports",
      "use/shortcuts",
      "use/ai",
      "use/telegram",
      "use/mobile-sync",
      "use/billing"
    ]
  },
  {
    "title": "Integrate",
    "items": [
      "integrate/rest-api",
      {
        "dir": "integrate/rest"
      },
      "integrate/cli",
      {
        "dir": "integrate/cli"
      },
      "integrate/mcp-stdio",
      "integrate/mcp-remote",
      {
        "dir": "integrate/mcp"
      },
      "integrate/sync-protocol",
      "integrate/export-format"
    ]
  },
  {
    "title": "Operate",
    "items": [
      "operate/environment",
      "operate/security",
      "operate/roles-workspaces"
    ]
  },
  {
    "title": "Contribute",
    "items": [
      "contribute/architecture",
      "contribute/release-train",
      "contribute/how-epics-work",
      "contribute/tests"
    ]
  },
  {
    "title": "Changelog",
    "items": [
      "changelog",
      {
        "dir": "changelog"
      }
    ]
  }
];

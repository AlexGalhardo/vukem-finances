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
      "index"
    ]
  }
];

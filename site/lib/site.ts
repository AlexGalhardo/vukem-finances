export const locales = ["en", "pt-br", "es"] as const;
export type Locale = (typeof locales)[number];

export const htmlLang: Record<Locale, string> = { en: "en", "pt-br": "pt-BR", es: "es" };
export const ogLocale: Record<Locale, string> = { en: "en_US", "pt-br": "pt_BR", es: "es_ES" };
export const localeNames: Record<Locale, string> = { en: "English", "pt-br": "Português", es: "Español" };

export const repoUrl = "https://github.com/AlexGalhardo/vukem-finances";
export const releasesUrl = `${repoUrl}/releases`;
export const ghcrImage = "ghcr.io/alexgalhardo/vukem-finances";
export const dockerHubImage = "alexgalhardo/vukem-finances";

// Paths on this site; the documentation itself is written by epic 23.
export const docsPath = "/docs/";
export const dockerDocsPath = "/docs/run/docker/";

export const siteOrigin = new URL(process.env.SITE_URL || "https://finances.vukem.dev");
export const basePath = siteOrigin.pathname.replace(/\/$/, "");

// The terms are a template until the author approves them.
export const termsApproved = process.env.LANDING_TERMS_APPROVED === "true";

/** Path of the landing page of a locale, with the base path and a trailing slash. */
export function localePath(locale: Locale): string {
  return `${basePath}${locale === "en" ? "/" : `/${locale}/`}`;
}

export function absolute(path: string): string {
  return new URL(path, siteOrigin.origin).href;
}

// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import { createI18n } from "vue-i18n";

import messages from "virtual:cherrypi-i18n-messages";

export const LOCALES = ["en", "ru"] as const;
export type Locale = (typeof LOCALES)[number];

export const DEFAULT_LOCALE: Locale = "en";

const STORAGE_KEY = "cherrypi.locale";

export const LOCALE_NAMES: Record<Locale, string> = {
  en: "English",
  ru: "Русский",
};

export function isLocale(value: unknown): value is Locale {
  return (
    typeof value === "string" && (LOCALES as readonly string[]).includes(value)
  );
}

export function detectBrowserLocale(): Locale {
  const languages =
    typeof navigator === "undefined"
      ? []
      : navigator.languages?.length > 0
        ? navigator.languages
        : [navigator.language];
  for (const tag of languages) {
    const base = tag.toLowerCase().split("-")[0];
    if (isLocale(base)) return base;
  }
  return DEFAULT_LOCALE;
}

export function cachedLocale(): Locale | undefined {
  try {
    const value = localStorage.getItem(STORAGE_KEY);
    return isLocale(value) ? value : undefined;
  } catch {
    return undefined;
  }
}

function initialLocale(): Locale {
  return cachedLocale() ?? detectBrowserLocale();
}

export const PLURAL_RULES: Record<
  string,
  (choice: number, choicesLength: number) => number
> = {
  ru(choice, choicesLength) {
    const n = Math.abs(choice);
    if (n % 10 === 1 && n % 100 !== 11) return 0;
    if (n % 10 >= 2 && n % 10 <= 4 && (n % 100 < 12 || n % 100 > 14)) return 1;
    return Math.min(2, choicesLength - 1);
  },
};

export const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: initialLocale(),
  fallbackLocale: DEFAULT_LOCALE,
  pluralRules: PLURAL_RULES,

  messages: messages as Record<string, never>,

  missingWarn: import.meta.env.DEV,
  fallbackWarn: import.meta.env.DEV,
  warnHtmlMessage: false,
});

export function currentLocale(): Locale {
  return i18n.global.locale.value as Locale;
}

export function applyLocale(locale: Locale): void {
  i18n.global.locale.value = locale;
  if (typeof document !== "undefined") {
    document.documentElement.lang = locale;
  }
  try {
    localStorage.setItem(STORAGE_KEY, locale);
  } catch {}
}

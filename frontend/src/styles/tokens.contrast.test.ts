import { readFileSync, readdirSync, statSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, relative } from "node:path";
import { describe, expect, it } from "vitest";

/**
 * Kontras token.
 *
 * `DESIGN.md` §3 dan `tokens.css` menyatakan angka kontras untuk setiap pasangan
 * teks/latar. Test ini **menghitungnya sendiri** dari berkas token, bukan
 * mempercayai angka di dokumen: bila sebuah token diubah tanpa alasan tertulis,
 * test gagal dan dokumennya harus diperbaiki pada sesi yang sama.
 *
 * Aturan yang diuji adalah WCAG 2.2 AA: teks 4,5:1; batas kontrol dan focus ring
 * (non-teks) 3:1 (SC 1.4.3 dan 1.4.11).
 */

const here = dirname(fileURLToPath(import.meta.url));
const tokensPath = join(here, "tokens.css");
const css = readFileSync(tokensPath, "utf8");

/** Bagian berkas CSS menjadi peta `nama variabel -> nilai`. */
function declarations(block: string): Record<string, string> {
  const result: Record<string, string> = {};
  for (const match of block.matchAll(/(--[a-z0-9-]+)\s*:\s*([^;]+);/gi)) {
    const name = match[1];
    const value = match[2];
    if (name && value) result[name] = value.trim();
  }
  return result;
}

/**
 * Memecah CSS menjadi blok `{ selector, body }`.
 *
 * Teks sebelum `{` bukan selalu selektor: pernyataan at-rule yang diakhiri `;`
 * (`@import`, `@custom-variant`) dan komentar ikut terbawa. Karena itu komentar
 * dibuang lebih dulu, lalu prelude dipotong pada `;` terakhir — sisa setelahnya
 * barulah selektor blok. Tanpa ini, blok `@theme` terbaca sebagai
 * `@import ...; @custom-variant ...; @theme` dan seluruh palet primitif tidak
 * pernah masuk peta token (pernah terjadi, dan test ini gagal karena itu).
 */
function blocksOf(source: string): { selector: string; body: string }[] {
  const withoutComments = source.replace(/\/\*[\s\S]*?\*\//g, "");
  return [...withoutComments.matchAll(/([^{}]+)\{([^{}]*)\}/g)].map((match) => {
    const prelude = (match[1] ?? "").trim();
    return {
      selector: prelude.slice(prelude.lastIndexOf(";") + 1).trim(),
      body: match[2] ?? "",
    };
  });
}

const blocks = blocksOf(css);

function themeMap(theme: "light" | "dark"): Record<string, string> {
  const map: Record<string, string> = {};
  for (const block of blocks) {
    const isLight = /:root|\[data-theme="light"\]/.test(block.selector);
    const isDark = /\[data-theme="dark"\]/.test(block.selector);
    // Blok `@theme` memuat nilai dasar (versi terang) dan primitif palet.
    const isBase = block.selector.startsWith("@theme");
    if (!isBase && !isLight && !isDark) continue;
    if (isDark && theme === "light") continue;
    Object.assign(map, declarations(block.body));
  }
  return map;
}

function resolve(map: Record<string, string>, name: string, depth = 0): string {
  const value = map[name];
  if (value === undefined) throw new Error(`token ${name} tidak ada di tokens.css`);
  const reference = /^var\((--[a-z0-9-]+)\)$/i.exec(value);
  if (reference && depth < 8) return resolve(map, reference[1] as string, depth + 1);
  return value;
}

function channel(value: number): number {
  return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
}

function luminance(hex: string): number {
  const clean = hex.trim().replace("#", "");
  const full =
    clean.length === 3
      ? clean
          .split("")
          .map((character) => character + character)
          .join("")
      : clean;
  const [r, g, b] = [0, 2, 4].map((offset) =>
    channel(parseInt(full.slice(offset, offset + 2), 16) / 255),
  ) as [number, number, number];
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(foreground: string, background: string): number {
  const a = luminance(foreground);
  const b = luminance(background);
  const [light, dark] = a > b ? [a, b] : [b, a];
  return (light + 0.05) / (dark + 0.05);
}

const themes = { light: themeMap("light"), dark: themeMap("dark") } as const;

const textTokens = ["--text", "--text-soft", "--text-muted"] as const;
const surfaces = ["--surface", "--surface-raised"] as const;
const statusTones = ["draft", "review", "revision", "approved", "rejected", "archived"] as const;

describe.each(["light", "dark"] as const)("tokens kontras — tema %s", (theme) => {
  const map = themes[theme];

  it.each(textTokens)("%s mencapai 4,5:1 di setiap permukaan", (token) => {
    for (const surface of surfaces) {
      const ratio = contrast(resolve(map, token), resolve(map, surface));
      expect(ratio, `${token} di atas ${surface} = ${ratio.toFixed(2)}:1`).toBeGreaterThanOrEqual(4.5);
    }
  });

  it("garis batas kontrol dan focus ring mencapai 3:1 (WCAG 1.4.11)", () => {
    for (const surface of surfaces) {
      const boundary = contrast(resolve(map, "--line-strong"), resolve(map, surface));
      expect(boundary, `--line-strong di atas ${surface}`).toBeGreaterThanOrEqual(3);
      const focus = contrast(resolve(map, "--accent-strong"), resolve(map, surface));
      expect(focus, `--accent-strong di atas ${surface}`).toBeGreaterThanOrEqual(3);
    }
  });

  it("tautan aksen dan teks bahaya terbaca sebagai teks", () => {
    for (const surface of surfaces) {
      expect(contrast(resolve(map, "--accent"), resolve(map, surface))).toBeGreaterThanOrEqual(4.5);
      expect(contrast(resolve(map, "--danger"), resolve(map, surface))).toBeGreaterThanOrEqual(4.5);
    }
  });

  it.each(statusTones)("badge status %s mencapai 4,5:1", (tone) => {
    const ink = resolve(map, `--color-status-${tone}-ink`);
    const surface = resolve(map, `--color-status-${tone}-surface`);
    const ratio = contrast(ink, surface);
    expect(ratio, `${tone} = ${ratio.toFixed(2)}:1`).toBeGreaterThanOrEqual(4.5);
  });
});

describe("angka kontras yang diumumkan DESIGN.md §2-§3", () => {
  const documented: [string, "light" | "dark", string, string, number][] = [
    ["teks utama", "light", "--text", "--surface", 16.33],
    ["teks sekunder", "light", "--text-soft", "--surface", 8.07],
    ["teks bantu", "light", "--text-muted", "--surface", 5.09],
    ["tautan aksen", "light", "--accent", "--surface", 7.15],
    ["teks utama", "dark", "--text", "--surface", 15.46],
    ["teks bantu", "dark", "--text-muted", "--surface", 6.93],
    ["focus ring", "dark", "--accent-strong", "--surface", 9.18],
  ];

  it.each(documented)("%s (%s) benar-benar %s/%s → %s:1", (_label, theme, fg, bg, expected) => {
    const ratio = contrast(resolve(themes[theme], fg), resolve(themes[theme], bg));
    expect(ratio).toBeCloseTo(expected, 1);
  });

  it("seluruh pasangan status versi gelap tetap di atas 9:1", () => {
    for (const tone of statusTones) {
      const ratio = contrast(
        resolve(themes.dark, `--color-status-${tone}-ink`),
        resolve(themes.dark, `--color-status-${tone}-surface`),
      );
      expect(ratio, tone).toBeGreaterThanOrEqual(9);
    }
  });
});

describe("aturan token", () => {
  it("tidak ada komponen yang menulis warna langsung", () => {
    const root = join(here, "..");
    const offenders: string[] = [];

    const walk = (directory: string): void => {
      for (const entry of readdirSync(directory)) {
        const path = join(directory, entry);
        if (statSync(path).isDirectory()) {
          walk(path);
          continue;
        }
        if (!/\.(tsx?|css)$/.test(entry)) continue;
        if (entry === "tokens.css" || entry.includes(".test.")) continue;
        const source = readFileSync(path, "utf8");
        if (/#[0-9a-f]{3,8}\b/i.test(source)) {
          offenders.push(relative(root, path));
        }
      }
    };

    walk(root);
    expect(offenders, `warna langsung ditemukan di: ${offenders.join(", ")}`).toEqual([]);
  });
});

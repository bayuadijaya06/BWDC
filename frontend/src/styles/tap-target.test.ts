import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

/**
 * Ambang sentuh harus berupa **kotak**, bukan hanya tinggi.
 *
 * `51-UX.md` §9 dan R-03 menuntut target 44x44px di layar sentuh (36x36px di
 * desktop, kepadatan alat kerja `DESIGN.md` §4). Utility `tap-target` semula
 * hanya menetapkan `min-height`, dan itu cukup untuk kontrol yang lebarnya
 * datang dari isian panjang — tetapi tidak untuk kontrol berlabel pendek: tab
 * sub-halaman "Tim" terukur **43x44px** pada 375px, satu piksel di bawah
 * ambangnya, dan ia satu-satunya yang setipis itu sehingga cacatnya menunggu
 * sampai ada label yang cukup pendek. Pengukuran itu ada di
 * `scripts/responsive-evidence.mjs`, dan test ini menjaga **sebabnya** supaya
 * tidak menunggu label pendek berikutnya.
 *
 * Test ini membaca `tokens.css` apa adanya, seperti test kontras membaca token:
 * yang dikunci adalah deklarasi di sumbernya, bukan angka yang dihafal di test.
 */

const here = dirname(fileURLToPath(import.meta.url));
const css = readFileSync(join(here, "tokens.css"), "utf8");

/** Isi blok `@utility tap-target { ... }`, kurung seimbang. */
function tapTargetBlock(source: string): string {
  const start = source.indexOf("@utility tap-target");
  if (start < 0) throw new Error("@utility tap-target tidak ditemukan di tokens.css");
  const open = source.indexOf("{", start);
  let depth = 0;
  for (let i = open; i < source.length; i += 1) {
    if (source[i] === "{") depth += 1;
    else if (source[i] === "}") {
      depth -= 1;
      if (depth === 0) return source.slice(open, i + 1);
    }
  }
  throw new Error("blok @utility tap-target tidak tertutup");
}

const block = tapTargetBlock(css);

describe("utility tap-target", () => {
  it("menetapkan tinggi minimum di kedua rentang lebar", () => {
    expect(block).toMatch(/min-height:\s*2\.75rem/);
    expect(block).toMatch(/@media\s*\(min-width:\s*64rem\)/);
    expect(block).toMatch(/min-height:\s*2\.25rem/);
  });

  it("menetapkan lebar minimum juga, sehingga ambangnya berupa kotak", () => {
    // Tanpa baris ini, satu label pendek (mis. tab "Tim") menghasilkan tombol
    // 43x44px: tinggi memenuhi ambang, lebarnya tidak, dan tidak ada yang
    // menangkapnya karena tidak ada test yang membaca ukuran dari CSS.
    expect(block).toMatch(/min-width:\s*2\.75rem/);
    expect(block).toMatch(/min-width:\s*2\.25rem/);
  });

  it("tidak mengunci lebar maupun tinggi tetap, supaya isi panjang tetap muat", () => {
    // Yang dilarang adalah deklarasi `width`/`height` **tetap** (tanpa awalan
    // `min-`/`max-`): mengunci tinggi membuat isian panjang terpotong, dan
    // mengunci lebar membuat kolom penyaring tidak dapat menyusut.
    const fixed = /(^|[;{\s])(width|height)\s*:/;
    expect(block).not.toMatch(fixed);
  });
});

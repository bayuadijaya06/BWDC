import type { RenderResult } from "@testing-library/react";
import axe from "axe-core";

/**
 * Pemeriksa aksesibilitas.
 *
 * Aturan `color-contrast` dimatikan di sini **bukan** karena diabaikan: jsdom
 * tidak menghitung tata letak, sehingga hasilnya selalu `incomplete`. Rasio
 * kontras diverifikasi terpisah dan angkanya dicatat di `DESIGN.md` §3, bukan
 * disimpulkan dari jsdom. Aturan lain (nama tombol, label form, peran ARIA,
 * urutan heading) tetap diperiksa.
 */
export async function runAxe(
  node: RenderResult["container"],
): Promise<string[]> {
  const results = await axe.run(node, {
    rules: { "color-contrast": { enabled: false } },
  });
  return results.violations.map((violation) => {
    const targets = violation.nodes.map((n) => n.target.join(" ")).join("; ");
    return `${violation.id}: ${violation.help} [${targets}]`;
  });
}

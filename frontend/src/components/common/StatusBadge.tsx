import type { StatusPresentation } from "@/types/status";

import { toneRuleClass, toneSurfaceClass } from "./tones";

/**
 * Badge status. Labelnya datang dari `50-FSD.md` §11 lewat `types/status.ts`,
 * jadi tidak ada nilai mentah kolom (`in_review`) yang pernah tampil ke
 * pengguna.
 *
 * Warna bukan satu-satunya pembawa makna (WCAG 1.4.1): badge selalu memuat
 * teksnya, dan bentuknya sengaja segi (bukan pil) supaya terbaca sebagai
 * stempel.
 */
export function StatusBadge({
  presentation,
}: {
  presentation: StatusPresentation;
}) {
  return (
    <span
      className={[
        "inline-flex items-center rounded-control border border-current/25 px-1.5 py-0.5",
        "text-12 font-medium whitespace-nowrap",
        toneSurfaceClass[presentation.tone],
      ].join(" ")}
    >
      {presentation.label}
    </span>
  );
}

/**
 * Penanda turunan `Overdue` (`50-FSD.md` §11.4). Ia bukan status, jadi ia
 * memakai **bentuk** yang berbeda dari badge: segitiga kecil + label, sehingga
 * baris yang punya dua penanda sekaligus (status + overdue) tetap terbaca.
 */
export function OverdueFlag() {
  return (
    <span
      className={[
        "inline-flex items-center gap-1 text-12 font-medium text-status-rejected-ink",
        "whitespace-nowrap",
      ].join(" ")}
    >
      <span
        aria-hidden="true"
        className={[
          "inline-block size-0 border-y-[4px] border-l-[6px] border-y-transparent",
          "border-l-status-rejected-ink",
        ].join(" ")}
      />
      Overdue
    </span>
  );
}

/**
 * Punggung rekam: garis tepi kiri yang warnanya mengikuti status
 * (`DESIGN.md` §6). Dipakai sebagai sel pertama baris tabel dan sebagai tepi
 * kepala halaman detail.
 */
export function Spine({
  tone,
  thickness = "row",
}: {
  tone: StatusPresentation["tone"];
  thickness?: "row" | "page";
}) {
  const width = thickness === "page" ? "w-1" : "w-[3px]";
  return (
    <span
      aria-hidden="true"
      className={[
        "block h-full min-h-9 shrink-0",
        width,
        toneRuleClass[tone],
      ].join(" ")}
    />
  );
}

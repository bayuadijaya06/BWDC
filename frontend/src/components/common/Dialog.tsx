import { useId, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";

import { useModalLayer } from "@/hooks/useModalLayer";

/**
 * Dialog modal (`50-FSD.md` §3.1: form buat project dan konfirmasi arsip).
 *
 * Yang dikerjakan di sini dan tidak diserahkan ke pemanggil, karena inilah
 * bagian yang paling mudah dilupakan:
 * - fokus dipindahkan ke dialog saat dibuka dan **dikembalikan** ke elemen
 *   pemicu saat ditutup (WCAG 2.4.3);
 * - Tab berputar di dalam dialog, sehingga fokus tidak pernah lepas ke halaman
 *   di belakangnya yang tetap terlihat;
 * - Escape menutup, dan halaman di belakang tidak dapat digulir.
 *
 * Dialog tidak ditutup dengan klik latar saat `pending`: menutup form di
 * tengah permintaan menyembunyikan hasilnya dan membuat pengguna mengirim dua
 * kali. Untuk keadaan itu latar sengaja diam.
 */
export function Dialog({
  title,
  description,
  onClose,
  children,
  footer,
  width = "max-w-[560px]",
}: {
  title: string;
  description?: ReactNode;
  onClose: () => void;
  children: ReactNode;
  footer?: ReactNode;
  width?: string;
}) {
  const titleId = useId();
  const descriptionId = useId();
  const panelRef = useRef<HTMLDivElement>(null);

  // Perilaku lapisan modal (fokus awal, jebakan Tab, Escape, kunci gulir, dan
  // pengembalian fokus) hidup di satu hook bersama laci menu `AppShell`, supaya
  // kedua lapisan tidak menyimpang satu sama lain.
  useModalLayer({ active: true, containerRef: panelRef, onClose });

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-ink-900/35 p-4 sm:items-center"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
        tabIndex={-1}
        className={[
          "w-full rounded-panel border border-line-strong bg-surface-raised",
          width,
        ].join(" ")}
      >
        {/* `div`, bukan `header`: `<header>` di luar sectioning element menjadi
            landmark `banner` kedua, dan axe menandainya
            (`landmark-no-duplicate-banner`). Panel boleh memakai `<header>`
            karena ia berada di dalam `<section>`. */}
        <div className="flex flex-col gap-0.5 border-b border-line px-4 py-3">
          <h2 id={titleId} className="text-16">
            {title}
          </h2>
          {description ? (
            <div id={descriptionId} className="text-12 text-text-muted">
              {description}
            </div>
          ) : null}
        </div>

        <div className="px-4 py-3.5">{children}</div>

        {footer ? (
          <div className="flex flex-wrap items-center justify-end gap-2 border-t border-line px-4 py-3">
            {footer}
          </div>
        ) : null}
      </div>
    </div>,
    document.body,
  );
}

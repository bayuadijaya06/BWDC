import type { ReactNode } from "react";

/**
 * Panel: satu blok rekam dengan judul kecil di atasnya.
 *
 * Batas panel dibuat **garis**, bukan bayangan (R-12). Sudut memakai
 * `rounded-panel` (3px) supaya halaman terasa seperti lembar arsip, bukan
 * deretan kartu bersudut bulat (R-11).
 */
interface PanelProps {
  title: string;
  /** Kalimat kecil di bawah judul, dipakai untuk menyebut sumber data. */
  note?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
}

export function Panel({ title, note, actions, children }: PanelProps) {
  return (
    <section className="rounded-panel border border-line bg-surface-raised">
      <header className="flex flex-wrap items-baseline justify-between gap-2 border-b border-line px-4 py-2.5">
        <div className="flex flex-col gap-0.5">
          <h2 className="text-16">{title}</h2>
          {note ? <div className="text-12 text-text-muted">{note}</div> : null}
        </div>
        {actions ? (
          <div className="flex items-center gap-2">{actions}</div>
        ) : null}
      </header>
      <div className="px-4 py-3.5">{children}</div>
    </section>
  );
}

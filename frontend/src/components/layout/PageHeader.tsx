import type { ReactNode } from "react";

import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import type { StatusPresentation } from "@/types/status";

import { Spine } from "../common/StatusBadge";

/**
 * Kepala halaman. Punggung rekam setebal 4px di tepi kiri adalah motif yang
 * sama dengan baris tabel, hanya pada skala halaman (`DESIGN.md` §6), dan ia
 * hanya muncul bila halaman itu memang mewakili satu rekam berstatus.
 */
export function PageHeader({
  title,
  description,
  presentation,
  actions,
}: {
  title: string;
  description?: ReactNode;
  presentation?: StatusPresentation;
  actions?: ReactNode;
}) {
  useDocumentTitle(title);

  return (
    <div className="flex items-stretch gap-3">
      {presentation ? (
        <Spine tone={presentation.tone} thickness="page" />
      ) : null}
      <div className="flex min-w-0 flex-1 flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          <h1 className="text-20 lg:text-26">{title}</h1>
          {description ? (
            <div className="text-13 text-text-muted">{description}</div>
          ) : null}
        </div>
        {actions ? (
          <div className="flex items-center gap-2">{actions}</div>
        ) : null}
      </div>
    </div>
  );
}

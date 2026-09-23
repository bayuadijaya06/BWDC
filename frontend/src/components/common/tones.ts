import type { StatusTone } from "@/types/status";

/**
 * Satu-satunya tempat nada status dipetakan ke kelas token. Badge, punggung
 * rekam, dan penanda timeline memakainya bersama, sehingga warna status tidak
 * pernah didefinisikan dua kali (aturan `50-FSD.md` §11: label dan nadanya satu
 * sumber).
 */
export const toneSurfaceClass: Record<StatusTone, string> = {
  draft: "bg-status-draft-surface text-status-draft-ink",
  review: "bg-status-review-surface text-status-review-ink",
  revision: "bg-status-revision-surface text-status-revision-ink",
  approved: "bg-status-approved-surface text-status-approved-ink",
  rejected: "bg-status-rejected-surface text-status-rejected-ink",
  archived: "bg-status-archived-surface text-status-archived-ink",
};

export const toneRuleClass: Record<StatusTone, string> = {
  draft: "bg-status-draft-ink",
  review: "bg-status-review-ink",
  revision: "bg-status-revision-ink",
  approved: "bg-status-approved-ink",
  rejected: "bg-status-rejected-ink",
  archived: "bg-status-archived-ink",
};

export const toneTextClass: Record<StatusTone, string> = {
  draft: "text-status-draft-ink",
  review: "text-status-review-ink",
  revision: "text-status-revision-ink",
  approved: "text-status-approved-ink",
  rejected: "text-status-rejected-ink",
  archived: "text-status-archived-ink",
};

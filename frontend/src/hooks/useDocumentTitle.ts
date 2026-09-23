import { useEffect } from "react";

/**
 * Menjaga `document.title` sesuai halaman. Berguna untuk riwayat peramban dan
 * pembaca layar, dan tidak menambah ketergantungan apa pun.
 */
export function useDocumentTitle(title: string): void {
  useEffect(() => {
    const previous = document.title;
    document.title = title === "Dashboard" ? "BWDCS" : `${title} · BWDCS`;
    return () => {
      document.title = previous;
    };
  }, [title]);
}

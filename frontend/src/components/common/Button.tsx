import type { ButtonHTMLAttributes, ReactNode } from "react";

/**
 * Tombol. Tidak ada varian "ghost dengan panah" dan tidak ada ikon dekoratif:
 * setiap tombol di aplikasi ini memanggil aksi nyata (R-26) dan teksnya
 * menyebut aksinya (`DESIGN.md` §1: hindari CTA generik, R-15).
 */

export type ButtonVariant = "primary" | "secondary" | "quiet" | "danger";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  /** Menampilkan keadaan menunggu sungguhan: tombol dinonaktifkan + aria-busy. */
  pending?: boolean;
  children: ReactNode;
}

const base =
  "inline-flex items-center justify-center gap-2 rounded-control font-medium " +
  "disabled:cursor-not-allowed disabled:opacity-55";

const variants: Record<ButtonVariant, string> = {
  primary:
    "bg-text text-surface-raised hover:opacity-90 shadow-sm font-semibold border border-text",
  secondary:
    "border border-line-strong bg-surface-raised text-text hover:bg-surface-hover",
  quiet: "text-text-soft hover:bg-surface-hover hover:text-text",
  danger:
    "border border-danger/45 bg-surface-raised text-danger hover:bg-surface-hover",
};

export function Button({
  variant = "secondary",
  pending = false,
  disabled,
  className,
  children,
  ...rest
}: ButtonProps) {
  const classes = [base, variants[variant], "tap-target px-3 text-13", className]
    .filter(Boolean)
    .join(" ");

  return (
    <button
      type="button"
      className={classes}
      disabled={disabled === true || pending}
      aria-busy={pending || undefined}
      {...rest}
    >
      {pending ? "Memproses" : children}
    </button>
  );
}

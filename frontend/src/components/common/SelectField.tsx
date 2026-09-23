import { useId, type SelectHTMLAttributes } from "react";

interface SelectFieldProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label: string;
  /** Keterangan tetap di bawah label, untuk aturan yang berlaku sebelum salah. */
  hint?: string;
  /** Pesan kesalahan. Ada isinya berarti field tidak valid. */
  error?: string | null;
  children: React.ReactNode;
}

/**
 * Field select: label, pilihan, dan pesan kesalahan inline.
 *
 * Dipisah dari `Field` karena `select` memiliki kebutuhan aksesibilitas dan
 * tata letak yang berbeda (membutuhkan `<option>` sebagai anak, bukan nilai).
 */
export function SelectField({ label, hint, error, className, children, ...rest }: SelectFieldProps) {
  const id = useId();
  const hintId = hint ? `${id}-hint` : undefined;
  const errorId = error ? `${id}-error` : undefined;
  const describedBy = [errorId, hintId].filter(Boolean).join(" ") || undefined;

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-13 font-medium text-text-soft">
        {label}
      </label>
      <select
        id={id}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        className={[
          "tap-target rounded-control border bg-surface-raised px-2.5 py-2 text-14 text-text",
          "disabled:opacity-55",
          error ? "border-danger" : "border-line-strong",
          className,
        ]
          .filter(Boolean)
          .join(" ")}
        {...rest}
      >
        {children}
      </select>
      {hint && !error ? (
        <p id={hintId} className="text-12 text-text-muted">
          {hint}
        </p>
      ) : null}
      {error ? (
        <p id={errorId} className="text-12 text-danger">
          {error}
        </p>
      ) : null}
    </div>
  );
}

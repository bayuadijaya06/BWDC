import {
  useId,
  type InputHTMLAttributes,
  type TextareaHTMLAttributes,
} from "react";

/**
 * Satu field form: label, input, dan pesan kesalahan inline.
 *
 * Kesalahan inline (bukan hanya toast) adalah syarat `51-UX.md` §7.2 dan
 * `DESIGN.md` §4. Pesan server dari `422` juga mengalir ke sini lewat prop
 * `error`, sehingga field yang salah ditunjuk persis.
 */
interface FieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "id"> {
  label: string;
  /** Keterangan tetap di bawah label, untuk aturan yang berlaku sebelum salah. */
  hint?: string;
  /** Pesan kesalahan. Ada isinya berarti field tidak valid. */
  error?: string | null;
}

export function Field({ label, hint, error, className, ...rest }: FieldProps) {
  const id = useId();
  const hintId = hint ? `${id}-hint` : undefined;
  const errorId = error ? `${id}-error` : undefined;
  const describedBy = [errorId, hintId].filter(Boolean).join(" ") || undefined;

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-13 font-medium text-text-soft">
        {label}
      </label>
      <input
        id={id}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        className={[
          "tap-target rounded-control border bg-surface-raised px-2.5 text-14 text-text",
          "placeholder:text-text-muted disabled:opacity-55",
          error ? "border-danger" : "border-line-strong",
          className,
        ]
          .filter(Boolean)
          .join(" ")}
        {...rest}
      />
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

interface TextareaFieldProps
  extends Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, "id"> {
  label: string;
  hint?: string;
  error?: string | null;
}

/**
 * Field teks panjang, dengan label dan pesan kesalahan yang sama seperti
 * `Field`. Dipisah karena `textarea` tidak menerima `type` dan tingginya diatur
 * `rows`, bukan kelas tinggi tetap.
 */
export function TextareaField({
  label,
  hint,
  error,
  className,
  ...rest
}: TextareaFieldProps) {
  const id = useId();
  const hintId = hint ? `${id}-hint` : undefined;
  const errorId = error ? `${id}-error` : undefined;
  const describedBy = [errorId, hintId].filter(Boolean).join(" ") || undefined;

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-13 font-medium text-text-soft">
        {label}
      </label>
      <textarea
        id={id}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        className={[
          "rounded-control border bg-surface-raised px-2.5 py-2 text-14 text-text",
          "placeholder:text-text-muted disabled:opacity-55",
          error ? "border-danger" : "border-line-strong",
          className,
        ]
          .filter(Boolean)
          .join(" ")}
        {...rest}
      />
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

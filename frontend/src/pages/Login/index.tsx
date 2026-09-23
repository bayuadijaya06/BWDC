import { useState, type FormEvent } from "react";
import { Navigate, useLocation, useNavigate } from "react-router";

import { Button } from "@/components/common/Button";
import { Field } from "@/components/common/Field";
import { ErrorMessage } from "@/components/common/States";
import { useAuthStore } from "@/store/auth";

/**
 * Halaman masuk.
 *
 * Yang diuji server dan tidak diulang di klien: username salah dan password
 * salah memakai pesan yang sama (`401 INVALID_CREDENTIALS`), jadi kesalahan
 * kredensial ditampilkan sebagai satu pesan, bukan per field. Yang diuji klien
 * hanyalah kelengkapan isian, dan itu ditampilkan inline (`51-UX.md` §7.2).
 */
export function LoginPage() {
  const status = useAuthStore((state) => state.status);
  const pending = useAuthStore((state) => state.pending);
  const error = useAuthStore((state) => state.error);
  const signIn = useAuthStore((state) => state.signIn);
  const clearError = useAuthStore((state) => state.clearError);
  const location = useLocation();
  const navigate = useNavigate();

  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [touched, setTouched] = useState(false);

  if (status === "authenticated") {
    const from = (location.state as { from?: string } | null)?.from ?? "/";
    return <Navigate to={from} replace />;
  }

  const usernameError =
    touched && username.trim() === "" ? "Username wajib diisi" : null;
  const passwordError =
    touched && password === "" ? "Password wajib diisi" : null;

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setTouched(true);
    clearError();
    if (username.trim() === "" || password === "") return;

    const ok = await signIn(username.trim(), password);
    if (ok) {
      const from = (location.state as { from?: string } | null)?.from ?? "/";
      navigate(from, { replace: true });
    }
  }

  return (
    <main className="flex min-h-dvh items-center justify-center bg-surface px-4 py-10">
      <div className="w-full max-w-sm">
        <div className="flex flex-col gap-1 pb-4">
          <p className="text-16 font-semibold tracking-tight">BWDCS</p>
          <p className="text-13 text-text-muted">
            Kendali dokumen dan alur kerja. Masuk memakai akun internal Anda.
          </p>
        </div>

        <form
          onSubmit={(event) => void onSubmit(event)}
          className="flex flex-col gap-4 rounded-panel border border-line bg-surface-raised px-4 py-5"
          noValidate
        >
          <Field
            label="Username"
            name="username"
            autoComplete="username"
            autoFocus
            value={username}
            error={usernameError}
            onChange={(event) => setUsername(event.target.value)}
          />
          <Field
            label="Password"
            name="password"
            type="password"
            autoComplete="current-password"
            value={password}
            error={passwordError}
            onChange={(event) => setPassword(event.target.value)}
          />

          {error ? <ErrorMessage error={error} /> : null}

          <Button
            type="submit"
            variant="primary"
            pending={pending}
            className="w-full"
          >
            Masuk
          </Button>
        </form>

        <p className="pt-3 text-12 text-text-muted">
          Ada berapa percobaan yang tersisa sebelum akun terkunci tidak
          diumumkan server (ADR-0022). Setelah ambangnya tercapai, akun terkunci
          sementara dan Administrator dapat membukanya lebih awal.
        </p>
      </div>
    </main>
  );
}

"use client";

// ─────────────────────────────────────────────
// components/auth/login-form.tsx
// Form login — UI only (tanpa server action / session).
// Fokus ke tampilan dulu; integrasi auth menyusul.
// ─────────────────────────────────────────────

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  ArrowRight,
  Eye,
  EyeOff,
  IdCard,
  Lock,
  LoaderCircle,
} from "lucide-react";
import { cn } from "@/lib/utils";

const labelClass =
  "block text-xs font-semibold tracking-wide uppercase text-text-muted mb-1.5";

function inputClass(hasError: boolean) {
  return cn(
    "w-full h-[42px] pl-10 pr-3 text-sm border rounded-lg bg-surface outline-none transition-colors",
    "placeholder:text-text-muted/70 focus:border-primary focus:ring-2 focus:ring-primary/15",
    hasError ? "border-status-danger-text" : "border-border-input",
  );
}

export function LoginForm() {
  const router = useRouter();
  const [showPassword, setShowPassword] = useState(false);
  const [showForgotNote, setShowForgotNote] = useState(false);
  const [pending, setPending] = useState(false);

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setPending(true);
    // TODO: hubungkan ke server action / API login
    // Sementara langsung redirect ke dashboard setelah delay singkat
    setTimeout(() => {
      setPending(false);
      router.push("/surat/baru");
    }, 800);
  }

  return (
    <form
      onSubmit={handleSubmit}
      noValidate
      className="w-full max-w-[480px] bg-surface border border-border rounded-xl shadow-sm p-8"
    >
      {/* NIP / NIK */}
      <div className="mb-5">
        <label htmlFor="nik" className={labelClass}>
          NIP / NIK Pegawai
        </label>
        <div className="relative">
          <IdCard
            size={16}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted"
            aria-hidden
          />
          <input
            id="nik"
            name="nik"
            type="text"
            autoComplete="username"
            autoFocus
            placeholder="Masukkan NIP / NIK"
            className={inputClass(false)}
          />
        </div>
      </div>

      {/* Password */}
      <div className="mb-4">
        <label htmlFor="password" className={labelClass}>
          Password
        </label>
        <div className="relative">
          <Lock
            size={16}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted"
            aria-hidden
          />
          <input
            id="password"
            name="password"
            type={showPassword ? "text" : "password"}
            autoComplete="current-password"
            placeholder="Masukkan kata sandi"
            className={cn(inputClass(false), "pr-10")}
          />
          <button
            type="button"
            onClick={() => setShowPassword((v) => !v)}
            aria-label={showPassword ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}
            className="absolute right-2.5 top-1/2 -translate-y-1/2 p-1 text-text-muted hover:text-text rounded"
          >
            {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
          </button>
        </div>
      </div>

      {/* Ingat saya + Lupa */}
      <div className="flex items-center justify-between mb-2">
        <label className="flex items-center gap-2 text-sm text-text-muted cursor-pointer select-none">
          <input
            type="checkbox"
            name="remember"
            className="size-4 accent-primary-dark rounded"
          />
          Ingat saya
        </label>
        <button
          type="button"
          onClick={() => setShowForgotNote((v) => !v)}
          aria-expanded={showForgotNote}
          className="text-sm font-medium text-text hover:underline"
        >
          Lupa Kata Sandi?
        </button>
      </div>
      {showForgotNote && (
        <p className="text-xs text-text-muted mb-4">
          Fitur reset kata sandi belum tersedia. Hubungi administrator.
        </p>
      )}

      <button
        type="submit"
        disabled={pending}
        className={cn(
          "mt-4 w-full h-[42px] flex items-center justify-center gap-2 rounded-lg text-sm font-semibold text-white",
          "bg-primary-dark hover:bg-primary transition-colors disabled:opacity-70 disabled:cursor-not-allowed",
        )}
      >
        {pending ? (
          <>
            <LoaderCircle size={16} className="animate-spin" aria-hidden />
            Memproses...
          </>
        ) : (
          <>
            Masuk ke Sistem
            <ArrowRight size={16} aria-hidden />
          </>
        )}
      </button>
    </form>
  );
}

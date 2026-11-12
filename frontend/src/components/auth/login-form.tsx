"use client";

// ─────────────────────────────────────────────
// components/auth/login-form.tsx
// Form login terintegrasi dengan backend.
// Fitur:
//   - Input NIK + password → loginAction (Server Action)
//   - Tampilkan pesan error dari backend
//   - "Akses Cepat Developer" untuk pilih role pengujian tanpa mengetik
//   - Redirect ke ?next= atau /surat/daftar setelah berhasil
// ─────────────────────────────────────────────

import { useState, useRef, useTransition } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import {
  ArrowRight,
  Eye,
  EyeOff,
  IdCard,
  Lock,
  LoaderCircle,
  AlertCircle,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { loginAction, devLoginAction } from "@/app/actions/auth";
import { DevLoginPanel } from "@/components/auth/dev-login-panel";

// Panel dev hanya untuk pengembangan; otomatis disembunyikan di production.
const SHOW_DEV_PANEL = process.env.NODE_ENV !== "production";
const labelClass =
  "block text-xs font-semibold tracking-wide uppercase text-text-muted mb-1.5";

function inputClass(hasError: boolean) {
  return cn(
    "w-full h-[42px] pl-10 pr-3 text-sm border rounded-lg bg-surface outline-none transition-colors",
    "placeholder:text-text-muted/70 focus:border-primary focus:ring-2 focus:ring-primary/15",
    hasError ? "border-status-danger-text ring-1 ring-status-danger-text/30" : "border-border-input"
  );
}

export function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const nextPath = searchParams.get("next") ?? "/dashboard";

  const [nikInput, setNikInput] = useState("");
  const passwordInputRef = useRef<HTMLInputElement>(null);

  const [showPassword, setShowPassword] = useState(false);
  const [showForgotNote, setShowForgotNote] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [devLoading, setDevLoading] = useState<string | null>(null);

  const [isPending, startTransition] = useTransition();

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setErrorMsg(null);
    const form = e.currentTarget;
    const nik = (form.elements.namedItem("nik") as HTMLInputElement).value.trim();
    const password = (form.elements.namedItem("password") as HTMLInputElement).value;

    if (!nik || !password) {
      setErrorMsg("NIK dan password tidak boleh kosong.");
      return;
    }

    startTransition(async () => {
      const result = await loginAction(nik, password);
      if (result.success) {
        router.push(nextPath);
        router.refresh(); // refresh agar server components baca cookie baru
      } else {
        setErrorMsg(result.error ?? "Login gagal.");
      }
    });
  }

  async function handleDevLogin(nik: string, label: string) {
    setDevLoading(nik);
    setErrorMsg(null);
    const result = await devLoginAction(nik);
    setDevLoading(null);
    if (result.success) {
      router.push(nextPath);
      router.refresh();
    } else {
      setErrorMsg(
        result.error ?? `Login gagal sebagai ${label}. Pastikan backend berjalan dan database telah di-seed.`
      );
    }
  }

  function handleFillNik(nik: string) {
    setNikInput(nik);
    setErrorMsg(null);
    if (passwordInputRef.current) {
      passwordInputRef.current.focus();
    }
  }

  return (
    <div className="w-full max-w-120 space-y-4">
      {/* ── Form Utama ─────────────────────────── */}
      <form
        onSubmit={handleSubmit}
        noValidate
        className="bg-surface border border-border rounded-xl shadow-sm p-8"
      >
        {/* Error banner */}
        {errorMsg && (
          <div className="flex items-start gap-2.5 bg-status-danger-bg border border-status-danger-text/30 text-status-danger-text text-sm rounded-lg px-3.5 py-3 mb-5">
            <AlertCircle size={16} className="shrink-0 mt-0.5" aria-hidden />
            <span>{errorMsg}</span>
          </div>
        )}

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
              value={nikInput}
              onChange={(e) => setNikInput(e.target.value)}
              placeholder="Contoh: 2015047 (Andi Kurniawan)"
              className={inputClass(!!errorMsg)}
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
              ref={passwordInputRef}
              id="password"
              name="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              placeholder="Masukkan kata sandi"
              className={cn(inputClass(!!errorMsg), "pr-10")}
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
          id="btn-login-submit"
          disabled={isPending}
          className={cn(
            "mt-4 w-full h-10.5 flex items-center justify-center gap-2 rounded-lg text-sm font-semibold text-white",
            "bg-primary-dark hover:bg-primary transition-colors disabled:opacity-70 disabled:cursor-not-allowed"
          )}
        >
          {isPending ? (
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

      {/* ── Panel Akses Cepat Developer ─────────── */}
      {SHOW_DEV_PANEL && (
        <DevLoginPanel
          onDevLogin={handleDevLogin}
          onFillNik={handleFillNik}
          devLoading={devLoading}
        />
      )}
    </div>
  );
}

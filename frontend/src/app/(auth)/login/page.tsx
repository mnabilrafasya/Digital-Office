import { Suspense } from "react";
import type { Metadata } from "next";
import { MailCheck } from "lucide-react";
import { LoginForm } from "@/components/auth/login-form";
import { redirect } from "next/navigation";
import { getCurrentUser } from "@/lib/auth";

export const metadata: Metadata = {
  title: "Masuk — Digital Office",
  description: "Masuk ke Sistem Persuratan Digital Office Perusahaan Pupuk.",
};

export default async function LoginPage() {
  // Redirect hanya bila sesi VALID (cookie basi tidak boleh, kalau tidak jadi loop).
  if (await getCurrentUser()) redirect("/surat/daftar");
  return (
    <>
      <div className="flex flex-col items-center text-center mb-6">
        <div className="w-12 h-12 rounded-xl bg-primary-dark flex items-center justify-center mb-4 shadow-md">
          <MailCheck size={22} className="text-white" aria-hidden />
        </div>
        <h1 className="text-2xl font-bold text-primary-dark">Digital Office</h1>
        <p className="text-sm text-text-muted mt-1">
          Sistem Persuratan Digital Perusahaan Pupuk
        </p>
      </div>
      {/* Suspense wajib karena LoginForm memakai useSearchParams() */}
      <Suspense
        fallback={
          <div className="w-full max-w-120 h-64 bg-surface border border-border rounded-xl animate-pulse" />
        }
      >
        <LoginForm />
      </Suspense>
    </>
  );
}

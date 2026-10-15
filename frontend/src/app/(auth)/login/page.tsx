import type { Metadata } from "next";
import { MailCheck } from "lucide-react";
import { LoginForm } from "@/components/auth/login-form";

export const metadata: Metadata = {
  title: "Masuk — Digital Office",
  description: "Masuk ke Sistem Persuratan Digital Office Perusahaan Pupuk.",
};

export default function LoginPage() {
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
      <LoginForm />
    </>
  );
}

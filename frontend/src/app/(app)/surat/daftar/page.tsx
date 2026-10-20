import { Suspense } from "react";
import {
  DaftarSuratView,
  DaftarSuratSkeleton,
} from "@/components/daftar-surat/daftar-surat-view";

// ─────────────────────────────────────────────
// app/(app)/surat/daftar/page.tsx
// Server Component page wrapper.
// Membungkus client view yang menggunakan useSearchParams()
// di dalam Suspense boundary agar prerendering tidak terblokir.
// ─────────────────────────────────────────────

export default function DaftarSuratPage() {
  return (
    <Suspense fallback={<DaftarSuratSkeleton />}>
      <DaftarSuratView />
    </Suspense>
  );
}

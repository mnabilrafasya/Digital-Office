import { Suspense } from "react";
import { Construction } from "lucide-react";

async function PageTitle({
  params,
}: {
  params: Promise<{ slug: string[] }>;
}) {
  const { slug } = await params;
  const title = slug[slug.length - 1]?.replace(/-/g, " ") ?? "Halaman";

  return <h1 className="text-xl font-bold text-text capitalize">{title}</h1>;
}

// Penampung sementara untuk semua menu sidebar yang halamannya belum dibuat.
// Hapus file ini setelah semua halaman ada.
export default function PlaceholderPage({
  params,
}: {
  params: Promise<{ slug: string[] }>;
}) {
  return (
    <div className="flex flex-col items-center justify-center text-center py-24">
      <div className="w-12 h-12 rounded-xl bg-panel flex items-center justify-center mb-4">
        <Construction size={22} className="text-text-muted" aria-hidden />
      </div>
      <Suspense fallback={<div className="h-7 w-32 bg-slate-200 animate-pulse rounded mx-auto" />}>
        <PageTitle params={params} />
      </Suspense>
      <p className="text-sm text-text-muted mt-1">Halaman ini belum tersedia.</p>
    </div>
  );
}

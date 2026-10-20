// ─────────────────────────────────────────────
// components/daftar-surat/summary-cards.tsx
// Empat kartu ringkasan di bagian atas halaman Daftar Surat
// ─────────────────────────────────────────────

import { FileText, AlertCircle, Clock, CheckCircle2 } from "lucide-react";

interface Counts {
  ALL: number;
  DRAFT: number;
  IN_REVIEW: number;
  RETURNED: number;
  IN_APPROVAL: number;
  SENT: number;
  CANCELLED: number;
}

interface Props {
  counts: Counts;
  loading?: boolean;
}

interface CardData {
  label: string;
  value: number | string;
  sub?: string;
  icon: React.ElementType;
  colorClass: string;
  bgClass: string;
}

export function SummaryCards({ counts, loading }: Props) {
  const cards: CardData[] = [
    {
      label: "Total Naskah Anda",
      value: counts.ALL,
      icon: FileText,
      colorClass: "text-primary",
      bgClass: "bg-primary/10",
    },
    {
      label: "Perlu Tindakan (Revisi)",
      value: counts.RETURNED,
      sub: "Segera Tanggapi",
      icon: AlertCircle,
      colorClass: "text-status-danger-text",
      bgClass: "bg-status-danger-bg",
    },
    {
      label: "Proses Review & Approval",
      value: counts.IN_REVIEW + counts.IN_APPROVAL,
      sub: `${counts.IN_REVIEW} Review • ${counts.IN_APPROVAL} Approval`,
      icon: Clock,
      colorClass: "text-status-warn-text",
      bgClass: "bg-status-warn-bg",
    },
    {
      label: "Disetujui",
      value: counts.SENT,
      icon: CheckCircle2,
      colorClass: "text-status-success-text",
      bgClass: "bg-status-success-bg",
    },
  ];

  if (loading) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3 mb-5">
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="bg-surface rounded-xl border border-border p-4 animate-pulse"
          >
            <div className="h-3 w-28 bg-panel rounded mb-3" />
            <div className="h-8 w-12 bg-panel rounded mb-2" />
            <div className="h-3 w-20 bg-panel rounded" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3 mb-5">
      {cards.map((card) => {
        const Icon = card.icon;
        return (
          <div
            key={card.label}
            className="bg-surface rounded-xl border border-border p-4 flex items-start justify-between gap-3 hover:shadow-sm transition-shadow"
          >
            <div className="min-w-0">
              <p className="text-xs text-text-muted font-medium leading-tight mb-1">
                {card.label}
              </p>
              <p className="text-3xl font-bold text-text leading-none mb-1">
                {card.value}
              </p>
              {card.sub && (
                <p className="text-xs text-text-muted">{card.sub}</p>
              )}
            </div>
            <div
              className={`shrink-0 w-10 h-10 rounded-lg flex items-center justify-center ${card.bgClass}`}
            >
              <Icon size={18} className={card.colorClass} />
            </div>
          </div>
        );
      })}
    </div>
  );
}

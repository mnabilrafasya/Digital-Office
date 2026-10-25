"use client";

// ─────────────────────────────────────────────
// components/shell/sidebar.tsx
// Sidebar navigasi — mengikuti menu di PROJECT_CONTEXT bagian 8
// Jangan tambah menu yang tidak ada di spec
// ─────────────────────────────────────────────

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  User,
  FilePlus,
  FileText,
  ClipboardCheck,
  CheckSquare,
  Database,
  Archive,
  ArchiveX,
  Shield,
  Inbox,
  Bell,
  BarChart3,
  Search,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useState, Suspense } from "react";

interface NavItem {
  label: string;
  href?: string;
  icon: React.ReactNode;
  badge?: number;
  children?: NavItem[];
}

const NAV_ITEMS: NavItem[] = [
  {
    label: "Dashboard",
    href: "/dashboard",
    icon: <LayoutDashboard size={16} />,
  },
  {
    label: "Profil",
    icon: <User size={16} />,
    children: [
      { label: "Edit Profil", href: "/profil/edit", icon: <User size={14} /> },
    ],
  },
  {
    label: "Penciptaan Surat",
    icon: <FilePlus size={16} />,
    children: [
      { label: "Buat Surat Baru",     href: "/surat/baru",          icon: <FilePlus size={14} /> },
      { label: "Buat SP/ASP",         href: "/surat/sp-asp",        icon: <FileText size={14} /> },
      { label: "Daftar Surat",        href: "/surat/daftar",        icon: <FileText size={14} /> },
      { label: "Menunggu Review",     href: "/surat/review",        icon: <ClipboardCheck size={14} /> },
      { label: "Menunggu Approval",   href: "/surat/approval",      icon: <CheckSquare size={14} /> },
    ],
  },
  {
    label: "Master Data",
    href: "/master-data",
    icon: <Database size={16} />,
  },
  {
    label: "Manajemen Arsip Aktif",
    href: "/arsip/aktif",
    icon: <Archive size={16} />,
  },
  {
    label: "Manajemen Arsip Inaktif",
    href: "/arsip/inaktif",
    icon: <ArchiveX size={16} />,
  },
  {
    label: "Permission Document",
    href: "/permission",
    icon: <Shield size={16} />,
  },
  {
    label: "Inbox",
    href: "/inbox",
    icon: <Inbox size={16} />,
    badge: 5,
  },
  {
    label: "Notifikasi",
    href: "/notifikasi",
    icon: <Bell size={16} />,
    badge: 3,
  },
  {
    label: "Rekap",
    href: "/rekap",
    icon: <BarChart3 size={16} />,
  },
  {
    label: "Pencarian",
    href: "/pencarian",
    icon: <Search size={16} />,
  },
];

function NavGroup({ item, depth = 0 }: { item: NavItem; depth?: number }) {
  const pathname = usePathname();
  const isActive = item.href
    ? pathname === item.href || pathname.startsWith(item.href + "/")
    : item.children?.some(
        (c) => c.href && (pathname === c.href || pathname.startsWith(c.href + "/")),
      );

  const [open, setOpen] = useState(isActive ?? false);

  if (item.children) {
    return (
      <div>
        <button
          onClick={() => setOpen((v) => !v)}
          className={cn(
            "w-full flex items-center justify-between gap-2 px-3 py-2 rounded-md text-sm transition-colors",
            "text-slate-300 hover:text-white hover:bg-white/10",
            isActive && "text-white bg-white/10",
          )}
        >
          <span className="flex items-center gap-2">
            {item.icon}
            <span>{item.label}</span>
          </span>
          {open ? <ChevronDown size={13} /> : <ChevronRight size={13} />}
        </button>
        {open && (
          <div className="mt-0.5 ml-3 border-l border-white/10 pl-3 space-y-0.5">
            {item.children.map((child) => (
              <NavGroup key={child.label} item={child} depth={depth + 1} />
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <Link
      href={item.href ?? "#"}
      className={cn(
        "flex items-center justify-between gap-2 px-3 py-2 rounded-md text-sm transition-colors",
        "text-slate-300 hover:text-white hover:bg-white/10",
        isActive && "text-white bg-white/15 font-medium",
      )}
    >
      <span className="flex items-center gap-2">
        {item.icon}
        <span>{item.label}</span>
      </span>
      {item.badge != null && item.badge > 0 && (
        <span className="bg-accent text-white text-[10px] font-bold rounded-full min-w-[18px] h-[18px] flex items-center justify-center px-1">
          {item.badge}
        </span>
      )}
    </Link>
  );
}

export function Sidebar() {
  return (
    <aside className="fixed inset-y-0 left-0 w-56 bg-primary-dark flex flex-col z-30 overflow-hidden">
      {/* Logo */}
      <div className="px-4 py-4 border-b border-white/10 shrink-0">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 rounded bg-primary flex items-center justify-center">
            <span className="text-white font-black text-sm">D</span>
          </div>
          <div>
            <div className="text-white font-bold text-sm leading-tight">Digital Office</div>
            <div className="text-slate-400 text-[10px] leading-tight">Sistem Persuratan</div>
          </div>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto px-2 py-3 space-y-0.5">
        <Suspense fallback={null}>
          {NAV_ITEMS.map((item) => (
            <NavGroup key={item.label} item={item} />
          ))}
        </Suspense>
      </nav>

      {/* Footer */}
      <div className="px-4 py-3 border-t border-white/10 shrink-0">
        <p className="text-slate-500 text-[10px]">© 2026 PT Pupuk Sriwidjaja Palembang</p>
      </div>
    </aside>
  );
}

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
import { useState, useEffect, Suspense } from "react";
import { getInboxUnreadCount } from "@/lib/api/inbox";
import { getPendingTaskCounts } from "@/lib/api/letters";

type DynamicBadgeType = "inbox" | "review" | "approval";

interface NavItem {
  label: string;
  href?: string;
  icon: React.ReactNode;
  badge?: number;
  dynamicBadge?: DynamicBadgeType;
  requiredPermission?: string;
  children?: NavItem[];
}

interface DynamicBadgeCounts {
  inbox: number;
  review: number;
  approval: number;
}

function OrangeCountBadge({ count }: { count: number }) {
  if (count <= 0) return null;
  return (
    <span className="bg-accent text-white text-[10px] font-bold rounded-full min-w-4.5 h-4.5 flex items-center justify-center px-1 shrink-0">
      {count > 99 ? "99+" : count}
    </span>
  );
}

// ─── Nav items ───────────────────────────────

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
      {
        label: "Buat Surat Baru",
        href: "/surat/baru",
        icon: <FilePlus size={14} />,
        requiredPermission: "LETTER_CREATE",
      },
      {
        label: "Daftar Surat",
        href: "/surat/daftar",
        icon: <FileText size={14} />,
      },
      {
        label: "Menunggu Review",
        href: "/surat/review",
        icon: <ClipboardCheck size={14} />,
        dynamicBadge: "review",
        requiredPermission: "LETTER_REVIEW",
      },
      {
        label: "Menunggu Approval",
        href: "/surat/approval",
        icon: <CheckSquare size={14} />,
        dynamicBadge: "approval",
        requiredPermission: "LETTER_APPROVE",
      },
    ],
  },
  {
    label: "Master Data",
    href: "/master-data",
    icon: <Database size={16} />,
    requiredPermission: "TEMPLATE_CREATE",
  },
  {
    label: "Permission Document",
    href: "/permission",
    icon: <Shield size={16} />,
    requiredPermission: "AUDIT_VIEW",
  },
  {
    label: "Inbox",
    href: "/inbox",
    icon: <Inbox size={16} />,
    dynamicBadge: "inbox", // badge dari backend, bukan hardcode
  },
  {
    label: "Notifikasi",
    href: "/notifikasi",
    icon: <Bell size={16} />,
  },
  {
    label: "Rekap",
    href: "/rekap",
    icon: <BarChart3 size={16} />,
    requiredPermission: "AUDIT_VIEW",
  },
  {
    label: "Pencarian",
    href: "/pencarian",
    icon: <Search size={16} />,
  },
];

function filterNavItemsByPermissions(
  items: NavItem[],
  permissions?: string[],
): NavItem[] {
  if (!permissions) return items;
  const permSet = new Set(permissions);

  return items.reduce<NavItem[]>((acc, item) => {
    if (item.requiredPermission && !permSet.has(item.requiredPermission)) {
      return acc;
    }
    if (item.children) {
      const filteredChildren = filterNavItemsByPermissions(
        item.children,
        permissions,
      );
      if (filteredChildren.length === 0) return acc;
      acc.push({ ...item, children: filteredChildren });
      return acc;
    }
    acc.push(item);
    return acc;
  }, []);
}

// ─── Nav group ───────────────────────────────

function NavGroup({
  item,
  depth = 0,
  badgeCounts,
}: {
  item: NavItem;
  depth?: number;
  badgeCounts: DynamicBadgeCounts;
}) {
  const pathname = usePathname();
  const isActive = item.href
    ? pathname === item.href || pathname.startsWith(item.href + "/")
    : item.children?.some(
        (c) =>
          c.href && (pathname === c.href || pathname.startsWith(c.href + "/")),
      );

  const [open, setOpen] = useState(isActive ?? false);

  if (item.children) {
    const childrenBadgeSum = item.children.reduce((sum, child) => {
      const dyn = child.dynamicBadge ? badgeCounts[child.dynamicBadge] : 0;
      return sum + (child.badge ?? 0) + dyn;
    }, 0);

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
          <span className="flex items-center gap-1.5">
            {!open && childrenBadgeSum > 0 && (
              <OrangeCountBadge count={childrenBadgeSum} />
            )}
            {open ? <ChevronDown size={13} /> : <ChevronRight size={13} />}
          </span>
        </button>
        {open && (
          <div className="mt-0.5 ml-3 border-l border-white/10 pl-3 space-y-0.5">
            {item.children.map((child) => (
              <NavGroup
                key={child.label}
                item={child}
                depth={depth + 1}
                badgeCounts={badgeCounts}
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  const resolvedBadge = item.dynamicBadge
    ? badgeCounts[item.dynamicBadge]
    : (item.badge ?? 0);

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
      <OrangeCountBadge count={resolvedBadge} />
    </Link>
  );
}

// ─── Sidebar ─────────────────────────────────

export function Sidebar({ permissions }: { permissions?: string[] }) {
  const pathname = usePathname();
  const visibleItems = filterNavItemsByPermissions(NAV_ITEMS, permissions);
  const [badgeCounts, setBadgeCounts] = useState<DynamicBadgeCounts>({
    inbox: 0,
    review: 0,
    approval: 0,
  });

  useEffect(() => {
    let active = true;
    Promise.all([getInboxUnreadCount(), getPendingTaskCounts()])
      .then(([inboxCount, tasks]) => {
        if (!active) return;
        setBadgeCounts({
          inbox: inboxCount,
          review: tasks.review,
          approval: tasks.approval,
        });
      })
      .catch(() => {});
    return () => {
      active = false;
    };
  }, [pathname]);

  return (
    <aside className="fixed inset-y-0 left-0 w-56 bg-primary-dark flex flex-col z-30 overflow-hidden">
      {/* Logo */}
      <div className="px-4 py-4 border-b border-white/10 shrink-0">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 rounded bg-primary flex items-center justify-center">
            <span className="text-white font-black text-sm">D</span>
          </div>
          <div>
            <div className="text-white font-bold text-sm leading-tight">
              Digital Office
            </div>
            <div className="text-slate-400 text-[10px] leading-tight">
              Sistem Persuratan
            </div>
          </div>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto px-2 py-3 space-y-0.5">
        <Suspense fallback={null}>
          {visibleItems.map((item) => (
            <NavGroup
              key={item.label}
              item={item}
              badgeCounts={badgeCounts}
            />
          ))}
        </Suspense>
      </nav>

      {/* Footer */}
      <div className="px-4 py-3 border-t border-white/10 shrink-0">
        <p className="text-slate-500 text-[10px]">
          &copy; 2026 PT Pupuk Sriwidjaja Palembang
        </p>
      </div>
    </aside>
  );
}

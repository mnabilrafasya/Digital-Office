"use client";

import {
  Plus,
  Pencil,
  Power,
  Trash2,
  KeyRound,
  UserPlus,
  Search,
  RefreshCw,
  CheckCircle2,
  Database,
  Info,
  Inbox,
  X,
  RotateCcw,
  Crown,
  Shield,
  Stamp,
  Hash,
  Building2,
  Users,
  FileText,
} from "lucide-react";
import { useState, type FormEvent, type ReactNode } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import {
  masterRequest,
  valueAt,
  type MasterList,
  type MasterRow,
} from "@/lib/api/master-data";
import {
  display,
  rowId,
  loadChoices,
  passwordField,
  userFields,
  type Resource,
  type Field,
} from "./config";
import {
  MasterForm,
  buttonClass,
  primaryButtonClass,
  rowActionButtonClass,
  rowDangerButtonClass,
  inputClass,
} from "./master-form";

type Action = {
  kind: "edit" | "create" | "status" | "delete" | "password" | "admin";
  row?: MasterRow;
};

const STATUS_TABS = [
  { key: "", label: "Semua Status" },
  { key: "ACTIVE", label: "Aktif" },
  { key: "INACTIVE", label: "Nonaktif" },
] as const;

function initials(name: string): string {
  return name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");
}

function renderCell(
  resource: Resource,
  row: MasterRow,
  columnKey: string,
  meta: { ownUser: boolean; ownCompany: boolean; readOnly: boolean; companyCode: string },
): ReactNode {
  if (resource.key === "users") {
    if (columnKey === "name") {
      const name = String(row.name ?? "—");
      const email = String(row.email ?? "—");
      const nik = String(row.employeeNumber ?? "—");
      return (
        <div className="flex items-start gap-2.5 min-w-0">
          <div className="w-8 h-8 rounded-full bg-primary/10 text-primary font-bold text-xs flex items-center justify-center shrink-0 mt-0.5 border border-primary/15">
            {initials(name) || "U"}
          </div>
          <div className="min-w-0 space-y-1">
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="font-semibold text-text text-sm leading-snug">
                {name}
              </span>
              {meta.ownUser && (
                <span className="rounded-full bg-primary/10 text-primary px-2 py-0.5 text-[10px] font-semibold">
                  Akun Anda
                </span>
              )}
            </div>
            <div className="flex flex-wrap items-center gap-1.5 text-xs text-text-muted">
              <span className="font-mono text-[11px] bg-panel px-1.5 py-0.5 rounded border border-border text-text">
                NIK {nik}
              </span>
              <span className="truncate max-w-48" title={email}>
                {email}
              </span>
            </div>
          </div>
        </div>
      );
    }

    if (columnKey === "position.name") {
      const posName = String(valueAt(row, "position.name") ?? "—");
      const unitName = String(valueAt(row, "unit.name") ?? "—");
      const levelCode = String(valueAt(row, "position.level.code") ?? "");
      const isUnitHead = Boolean(valueAt(row, "position.isUnitHead"));
      const isOfficial = Boolean(valueAt(row, "position.isOfficial"));
      return (
        <div className="space-y-1 min-w-0">
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="font-medium text-text text-xs leading-snug">
              {posName}
            </span>
            {levelCode && (
              <span className="rounded bg-panel px-1.5 py-0.5 text-[10px] font-mono text-text-muted border border-border">
                {levelCode}
              </span>
            )}
          </div>
          <p className="text-xs text-text-muted leading-snug">{unitName}</p>
          {(isUnitHead || isOfficial) && (
            <div className="flex flex-wrap gap-1 pt-0.5">
              {isUnitHead && (
                <span className="inline-flex items-center gap-1 rounded-full bg-status-warn-bg text-status-warn-text px-2 py-0.5 text-[10px] font-semibold">
                  <Crown size={10} />
                  Pimpinan Unit
                </span>
              )}
              {isOfficial && (
                <span className="inline-flex items-center gap-1 rounded-full bg-status-info-bg text-status-info-text px-2 py-0.5 text-[10px] font-semibold">
                  <Shield size={10} />
                  Pejabat
                </span>
              )}
            </div>
          )}
        </div>
      );
    }

    if (columnKey === "role.name") {
      const roleCode = String(valueAt(row, "role.code") ?? "");
      const roleName = String(valueAt(row, "role.name") ?? "—");
      const tone =
        roleCode === "SUPERADMIN"
          ? "bg-primary/15 text-primary border-primary/25"
          : roleCode === "ADMIN"
            ? "bg-status-warn-bg text-status-warn-text border-status-warn-text/20"
            : "bg-panel text-text-muted border-border";
      return (
        <span
          className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-semibold whitespace-nowrap ${tone}`}
        >
          {roleName}
        </span>
      );
    }
  }

  if (resource.key === "companies") {
    if (columnKey === "name") {
      const code = String(row.code ?? "—");
      const name = String(row.name ?? "—");
      const city = String(row.city ?? "");
      return (
        <div className="flex items-start gap-2.5 min-w-0">
          <div className="px-2 py-1 rounded-lg bg-primary/10 text-primary font-mono font-bold text-xs border border-primary/20 shrink-0 mt-0.5">
            {code}
          </div>
          <div className="space-y-0.5 min-w-0">
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="font-semibold text-text text-sm leading-snug">
                {name}
              </span>
              {meta.ownCompany && (
                <span className="rounded-full bg-primary/10 text-primary px-2 py-0.5 text-[10px] font-semibold">
                  Perusahaan Aktif Anda
                </span>
              )}
            </div>
            <p className="text-xs text-text-muted">
              {city ? `Kedudukan: ${city}` : "Kota kedudukan belum diisi"}
            </p>
          </div>
        </div>
      );
    }

    if (columnKey === "letterNumberFormat") {
      const format = String(row.letterNumberFormat ?? "—");
      const watermark = row.watermarkText ? String(row.watermarkText) : null;
      return (
        <div className="space-y-1 min-w-0">
          <div className="inline-flex items-center gap-1.5 rounded-md bg-panel px-2 py-0.5 text-[11px] font-mono text-text border border-border break-all">
            <Hash size={11} className="text-primary shrink-0" />
            <span>{format}</span>
          </div>
          <div className="flex items-center gap-1.5 text-xs">
            <Stamp size={11} className="text-text-muted shrink-0" />
            {watermark ? (
              <span className="text-text font-medium">
                Watermark: <strong className="font-mono">{watermark}</strong>
              </span>
            ) : (
              <span className="text-text-muted italic">Tanpa watermark PDF</span>
            )}
          </div>
        </div>
      );
    }

    if (columnKey === "activeUnits") {
      return (
        <div className="flex flex-wrap gap-1 min-w-0">
          <span className="inline-flex items-center gap-1 rounded-md bg-panel px-2 py-0.5 text-xs text-text-muted border border-border whitespace-nowrap">
            <Building2 size={11} className="text-primary" />
            <strong className="text-text">{Number(row.activeUnits ?? 0)}</strong>{" "}
            unit
          </span>
          <span className="inline-flex items-center gap-1 rounded-md bg-panel px-2 py-0.5 text-xs text-text-muted border border-border whitespace-nowrap">
            <Users size={11} className="text-primary" />
            <strong className="text-text">{Number(row.activeUsers ?? 0)}</strong>{" "}
            user
          </span>
          <span className="inline-flex items-center gap-1 rounded-md bg-panel px-2 py-0.5 text-xs text-text-muted border border-border whitespace-nowrap">
            <FileText size={11} className="text-primary" />
            <strong className="text-text">{Number(row.letters ?? 0)}</strong>{" "}
            surat
          </span>
        </div>
      );
    }
  }

  if (resource.key === "units") {
    if (columnKey === "name") {
      const code = String(row.code ?? "—");
      const shortCode = code.length > 14 ? `${code.slice(0, 8)}…` : code;
      return (
        <div className="space-y-1 min-w-0">
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="font-semibold text-text text-sm leading-snug">
              {String(row.name ?? "—")}
            </span>
            <span className="rounded-md bg-primary/10 text-primary px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide">
              {String(row.unitType ?? "UNIT")}
            </span>
          </div>
          <p
            className="text-[11px] font-mono text-text-muted truncate max-w-44"
            title={`Kode Unit: ${code}`}
          >
            Kode: {shortCode}
          </p>
        </div>
      );
    }

    if (columnKey === "parentName") {
      return row.parentName ? (
        <span className="text-xs font-medium text-text leading-snug">
          {String(row.parentName)}
        </span>
      ) : (
        <span className="inline-flex rounded-md bg-panel px-2 py-0.5 text-[11px] text-text-muted border border-border whitespace-nowrap">
          Unit Puncak (Utama)
        </span>
      );
    }

    if (columnKey === "headName") {
      if (!row.headName && !row.headPosition) {
        return (
          <span className="text-xs text-text-muted italic">
            Belum ada pimpinan
          </span>
        );
      }
      return (
        <div className="space-y-0.5 min-w-0">
          <p className="text-xs font-semibold text-text leading-snug">
            {row.headName ? String(row.headName) : "Belum ada pejabat aktif"}
          </p>
          {Boolean(row.headPosition) && (
            <p className="text-[11px] text-text-muted leading-snug">
              {String(row.headPosition)}
            </p>
          )}
        </div>
      );
    }

    if (columnKey === "activeUsers") {
      return (
        <div className="flex flex-wrap gap-1 min-w-0">
          <span className="rounded-md bg-panel px-2 py-0.5 text-xs text-text-muted border border-border whitespace-nowrap">
            <strong className="text-text">
              {Number(row.activePositions ?? 0)}
            </strong>{" "}
            jabatan
          </span>
          <span className="rounded-md bg-panel px-2 py-0.5 text-xs text-text-muted border border-border whitespace-nowrap">
            <strong className="text-text">{Number(row.activeUsers ?? 0)}</strong>{" "}
            pegawai
          </span>
        </div>
      );
    }
  }

  if (resource.key === "positions") {
    if (columnKey === "name") {
      const isUnitHead = Boolean(row.isUnitHead);
      const isOfficial = Boolean(row.isOfficial);
      return (
        <div className="space-y-1 min-w-0">
          <p className="font-semibold text-text text-sm leading-snug">
            {String(row.name ?? "—")}
          </p>
          <div className="flex flex-wrap gap-1">
            {isUnitHead && (
              <span className="inline-flex items-center gap-1 rounded-full bg-status-warn-bg text-status-warn-text px-2 py-0.5 text-[10px] font-semibold">
                <Crown size={10} />
                Pimpinan Unit
              </span>
            )}
            {isOfficial ? (
              <span className="inline-flex items-center gap-1 rounded-full bg-status-info-bg text-status-info-text px-2 py-0.5 text-[10px] font-semibold">
                <Shield size={10} />
                Pejabat (Approve & Disposisi)
              </span>
            ) : (
              <span className="rounded-full bg-panel text-text-muted px-2 py-0.5 text-[10px]">
                Staf / Fungsional
              </span>
            )}
          </div>
        </div>
      );
    }

    if (columnKey === "unitName") {
      return (
        <span className="text-xs font-medium text-text leading-snug">
          {String(row.unitName ?? "—")}
        </span>
      );
    }

    if (columnKey === "level.name") {
      const levelName = String(valueAt(row, "level.name") ?? "—");
      const levelCode = String(valueAt(row, "level.code") ?? "");
      return (
        <div className="space-y-0.5">
          <span className="text-xs font-semibold text-text">{levelName}</span>
          {levelCode && (
            <span className="block font-mono text-[10px] text-text-muted">
              {levelCode}
            </span>
          )}
        </div>
      );
    }

    if (columnKey === "holders") {
      const holders = Array.isArray(row.holders) ? (row.holders as string[]) : [];
      if (holders.length === 0) {
        return (
          <span className="text-xs text-text-muted italic">
            Belum ada pemegang
          </span>
        );
      }
      return (
        <div className="flex flex-wrap gap-1 max-w-56">
          {holders.slice(0, 3).map((holder) => (
            <span
              key={holder}
              className="rounded-md bg-panel border border-border px-2 py-0.5 text-xs font-medium text-text"
            >
              {holder}
            </span>
          ))}
          {holders.length > 3 && (
            <span className="rounded-md bg-primary/10 text-primary px-2 py-0.5 text-xs font-semibold">
              +{holders.length - 3} lainnya
            </span>
          )}
        </div>
      );
    }
  }

  if (resource.key.startsWith("lookup-")) {
    if (columnKey === "code") {
      return (
        <span className="inline-flex font-mono text-xs font-bold rounded-md bg-primary/10 text-primary px-2.5 py-1 border border-primary/15">
          {String(row.code ?? "—")}
        </span>
      );
    }
    if (columnKey === "name") {
      return (
        <div className="space-y-1">
          <p className="font-semibold text-text text-sm">
            {String(row.name ?? "—")}
          </p>
          <span
            className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-semibold ${
              meta.readOnly
                ? "bg-panel text-text-muted border border-border"
                : "bg-status-info-bg text-status-info-text"
            }`}
          >
            {meta.readOnly
              ? "Standar Sistem (Global)"
              : `Khusus ${meta.companyCode}`}
          </span>
        </div>
      );
    }
    if (columnKey === "sortOrder") {
      return (
        <span className="inline-flex font-mono text-xs text-text-muted bg-panel px-2 py-0.5 rounded border border-border">
          #{String(row.sortOrder ?? 0)}
        </span>
      );
    }
  }

  return display(valueAt(row, columnKey));
}

export function ResourcePanel({
  resource,
  userId,
  companyCode,
  canManageSuperadmin = false,
}: {
  resource: Resource;
  userId: string;
  companyCode: string;
  canManageSuperadmin?: boolean;
}) {
  const cache = useQueryClient();
  const router = useRouter();
  const [page, setPage] = useState(1);
  const [q, setQ] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [status, setStatus] = useState("");
  const [unitId, setUnitId] = useState("");
  const [roleId, setRoleId] = useState("");
  const [action, setAction] = useState<Action | null>(null);
  const [notice, setNotice] = useState("");

  const supportsUnitFilter =
    resource.key === "positions" || resource.key === "users";
  const supportsRoleFilter = resource.key === "users";

  const unitsFilterQuery = useQuery({
    queryKey: ["master", "choices", "units", undefined],
    queryFn: () => loadChoices("units"),
    enabled: supportsUnitFilter,
  });

  const rolesFilterQuery = useQuery({
    queryKey: ["master", "choices", "roles", undefined],
    queryFn: () => loadChoices("roles"),
    enabled: supportsRoleFilter,
  });

  const params = new URLSearchParams({
    q,
    status,
    page: String(page),
    pageSize: "20",
  });
  if (supportsUnitFilter && unitId) params.set("unitId", unitId);
  if (supportsRoleFilter && roleId) params.set("roleId", roleId);

  const query = useQuery({
    queryKey: ["master", resource.key, q, status, unitId, roleId, page],
    queryFn: () => masterRequest<MasterList>(`${resource.path}?${params}`),
  });

  const rows = query.data?.data ?? [];
  const total = query.data?.meta?.total ?? rows.length;
  const pages = Math.max(1, Math.ceil(total / 20));
  const hasActiveFilter = Boolean(q || status || unitId || roleId);

  const pathFor = (row: MasterRow) =>
    `${resource.path}/${encodeURIComponent(rowId(resource, row))}`;

  async function save(values: MasterRow) {
    if (!action) return;
    const row = action.row;

    switch (action.kind) {
      case "create":
        await masterRequest(resource.path, "POST", values);
        break;
      case "edit":
        await masterRequest(pathFor(row!), "PUT", values);
        break;
      case "status":
        await masterRequest(`${pathFor(row!)}/status`, "PATCH", {
          active: !row!.isActive,
        });
        break;
      case "delete":
        await masterRequest(pathFor(row!), "DELETE");
        break;
      case "password":
        await masterRequest(`${pathFor(row!)}/reset-password`, "POST", values);
        break;
      case "admin": {
        if (!values.unitId && !values.unitName) {
          throw new Error("Pilih unit yang tersedia atau isi nama unit baru.");
        }
        const body = { ...values };
        if (body.unitId) {
          delete body.unitName;
        } else {
          delete body.unitId;
        }
        await masterRequest(`${pathFor(row!)}/admins`, "POST", body);
        break;
      }
    }

    setNotice("Perubahan berhasil disimpan.");
    await cache.invalidateQueries({ queryKey: ["master"] });
    if (resource.key === "users" && String(row?.id) === userId) {
      router.refresh();
    }
  }

  function search(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPage(1);
    setQ(searchInput.trim());
    setNotice("");
  }

  function resetFilters() {
    setSearchInput("");
    setQ("");
    setStatus("");
    setUnitId("");
    setRoleId("");
    setPage(1);
  }

  let fields: Field[] = resource.fields.filter(
    (field) => action?.kind === "create" || !field.createOnly,
  );
  let title = `${action?.kind === "create" ? "Tambah" : "Edit"} ${resource.label}`;
  let description: string | undefined;

  if (action?.kind === "password") {
    fields = [passwordField];
    title = `Reset Password — ${String(action.row?.name ?? "User")}`;
  }

  if (action?.kind === "status") {
    fields = [];
    title = action.row?.isActive ? "Nonaktifkan Data" : "Aktifkan Data";
    description = `${String(action.row?.name)} akan ${
      action.row?.isActive ? "dinonaktifkan" : "diaktifkan kembali"
    }. Data dan riwayat surat lama tetap tersimpan dengan aman.${
      resource.key === "companies"
        ? " Perusahaan yang nonaktif tidak dapat diakses oleh seluruh penggunanya."
        : ""
    }`;
  }

  if (action?.kind === "delete") {
    fields = [];
    title = `Hapus ${resource.label}`;
    description = `Hapus "${String(action.row?.name ?? action.row?.code ?? "")}" dari daftar ${resource.label.toLowerCase()}? Data yang dihapus tidak lagi tampil di master data (dan kode/NIK dapat dipakai ulang), namun riwayat surat serta audit log tetap terjaga di database.`;
  }

  if (action?.kind === "admin") {
    title = `Tambah Admin — ${action.row?.name}`;
    fields = [
      {
        key: "unitId",
        label: "Unit kerja yang tersedia",
        type: "select",
        choices: "units",
      },
      {
        key: "unitName",
        label: "Nama unit baru (bila belum memilih unit di atas)",
        max: 150,
        placeholder: "Contoh: Sekretariat Perusahaan",
      },
      {
        key: "positionName",
        label: "Nama jabatan administrator",
        required: true,
        max: 150,
        placeholder: "Contoh: Administrator Sistem",
      },
      ...userFields,
      passwordField,
    ];
  }

  return (
    <section className="space-y-5">
      {/* Section Header */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="font-bold text-base text-text">
              Daftar {resource.label}
            </h2>
            <span className="rounded-full bg-panel border border-border px-2.5 py-0.5 text-xs font-semibold text-text-muted">
              {total} data
            </span>
          </div>
          <p className="text-xs text-text-muted mt-0.5">
            Kelola data aktif maupun nonaktif. Seluruh perubahan dicatat pada
            audit log.
          </p>
        </div>

        {resource.fields.length > 0 && (
          <button
            type="button"
            className={primaryButtonClass}
            onClick={() => {
              setNotice("");
              setAction({ kind: "create" });
            }}
          >
            <Plus size={15} />
            Tambah {resource.label}
          </button>
        )}
      </div>

      {/* Underline Status Tabs (Consistent with /daftar-surat) */}
      <div className="flex items-center justify-between border-b border-border gap-3">
        <div
          role="tablist"
          aria-label="Filter status data"
          className="flex items-center gap-1 overflow-x-auto no-scrollbar"
        >
          {STATUS_TABS.map((tab) => {
            const isActive = status === tab.key;
            return (
              <button
                key={tab.key || "ALL"}
                type="button"
                role="tab"
                aria-selected={isActive}
                onClick={() => {
                  setStatus(tab.key);
                  setPage(1);
                }}
                className={`relative px-3.5 py-2.5 text-xs font-semibold transition-colors whitespace-nowrap ${
                  isActive
                    ? "text-primary"
                    : "text-text-muted hover:text-text"
                }`}
              >
                {tab.label}
                {isActive && (
                  <span className="absolute inset-x-0 bottom-0 h-0.5 bg-primary rounded-full" />
                )}
              </button>
            );
          })}
        </div>

        <button
          type="button"
          className="inline-flex items-center gap-1.5 text-xs font-medium text-text-muted hover:text-primary px-2 py-1.5 transition-colors shrink-0"
          onClick={() => query.refetch()}
          disabled={query.isFetching}
        >
          <RefreshCw
            size={13}
            className={query.isFetching ? "animate-spin text-primary" : ""}
          />
          <span className="hidden sm:inline">Muat ulang</span>
        </button>
      </div>

      {/* Search & Smart Filter Bar */}
      <div className="flex flex-wrap items-center gap-2.5 rounded-xl bg-panel/40 p-3 border border-border">
        <form onSubmit={search} className="flex gap-2 flex-1 min-w-60">
          <div className="relative flex-1">
            <Search
              size={15}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none"
            />
            <input
              name="q"
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              aria-label={`Cari ${resource.label}`}
              placeholder={`Cari ${resource.label.toLowerCase()}...`}
              className={`${inputClass} pl-9 pr-8`}
            />
            {searchInput && (
              <button
                type="button"
                aria-label="Hapus kata kunci"
                onClick={() => {
                  setSearchInput("");
                  if (q) {
                    setQ("");
                    setPage(1);
                  }
                }}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-muted hover:text-text"
              >
                <X size={14} />
              </button>
            )}
          </div>
          <button type="submit" className={`${buttonClass} bg-surface`}>
            Cari
          </button>
        </form>

        {supportsUnitFilter && (
          <select
            aria-label="Filter unit kerja"
            value={unitId}
            onChange={(event) => {
              setUnitId(event.target.value);
              setPage(1);
            }}
            className={`${inputClass} w-auto max-w-56`}
          >
            <option value="">Semua unit kerja</option>
            {(unitsFilterQuery.data ?? []).map((choice) => (
              <option key={choice.value} value={choice.value}>
                {choice.label}
              </option>
            ))}
          </select>
        )}

        {supportsRoleFilter && (
          <select
            aria-label="Filter role"
            value={roleId}
            onChange={(event) => {
              setRoleId(event.target.value);
              setPage(1);
            }}
            className={`${inputClass} w-auto max-w-44`}
          >
            <option value="">Semua role</option>
            {(rolesFilterQuery.data ?? []).map((choice) => (
              <option key={choice.value} value={choice.value}>
                {choice.label}
              </option>
            ))}
          </select>
        )}

        {hasActiveFilter && (
          <button
            type="button"
            onClick={resetFilters}
            className={`${buttonClass} bg-surface text-text-muted hover:text-text`}
          >
            <RotateCcw size={13} />
            Reset
          </button>
        )}
      </div>

      {/* Feedback Banner */}
      {notice && (
        <p
          role="status"
          className="flex items-center justify-between gap-2 rounded-xl bg-status-success-bg px-4 py-3 text-xs font-medium text-status-success-text"
        >
          <span className="flex items-center gap-2">
            <CheckCircle2 size={15} />
            {notice}
          </span>
          <button
            type="button"
            onClick={() => setNotice("")}
            aria-label="Tutup notifikasi"
            className="opacity-75 hover:opacity-100"
          >
            <X size={14} />
          </button>
        </p>
      )}

      {resource.key.startsWith("lookup-") && (
        <div className="flex items-start gap-2.5 p-3.5 bg-primary/5 border border-primary/15 rounded-xl text-xs text-text-muted leading-relaxed">
          <Info size={16} className="shrink-0 text-primary mt-0.5" />
          <p>
            Referensi berlabel{" "}
            <strong className="text-text">Standar Sistem (Global)</strong>{" "}
            berlaku lintas perusahaan dan dikunci agar tetap seragam. Anda dapat
            menambahkan kode <strong className="text-text">Khusus {companyCode}</strong>{" "}
            sesuai kebutuhan persuratan perusahaan Anda.
          </p>
        </div>
      )}

      {/* Table or Error/Skeleton State */}
      {query.isError ? (
        <div
          role="alert"
          className="rounded-xl bg-status-danger-bg p-4 text-sm text-status-danger-text flex items-center justify-between gap-3"
        >
          <span>{query.error.message}</span>
          <button
            type="button"
            className="underline font-semibold shrink-0"
            onClick={() => query.refetch()}
          >
            Coba lagi
          </button>
        </div>
      ) : query.isPending ? (
        <div
          role="status"
          aria-label="Memuat data..."
          className="rounded-xl border border-border overflow-hidden bg-surface"
        >
          <div className="bg-panel/70 px-4 py-3 border-b border-border flex gap-4">
            {resource.columns.map((col) => (
              <div
                key={col.key}
                className="h-3.5 w-28 rounded bg-border/70 animate-pulse"
              />
            ))}
          </div>
          <div className="divide-y divide-border">
            {[1, 2, 3, 4, 5].map((item) => (
              <div key={item} className="p-4 flex items-center justify-between gap-4">
                <div className="space-y-2 flex-1">
                  <div className="h-4 w-48 rounded bg-panel animate-pulse" />
                  <div className="h-3 w-32 rounded bg-panel/70 animate-pulse" />
                </div>
                <div className="h-6 w-20 rounded-full bg-panel animate-pulse" />
                <div className="h-8 w-28 rounded-lg bg-panel animate-pulse" />
              </div>
            ))}
          </div>
        </div>
      ) : (
        <div className="rounded-xl border border-border overflow-x-auto bg-surface">
          <table className="w-full text-sm text-left">
            <thead className="bg-panel/70 text-[11px] uppercase tracking-wider text-text-muted border-b border-border">
              <tr>
                {resource.columns.map((column) => (
                  <th key={column.key} className="px-3.5 py-3 font-semibold">
                    {column.label}
                  </th>
                ))}
                <th className="px-3.5 py-3 font-semibold">Status</th>
                <th className="px-3.5 py-3 font-semibold text-right">Tindakan</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {rows.map((row) => {
                const ownUser =
                  resource.key === "users" && String(row.id) === userId;
                const readOnly = Boolean(row.readOnly);
                const protectedUser =
                  resource.key === "users" &&
                  valueAt(row, "role.code") === "SUPERADMIN" &&
                  !canManageSuperadmin;
                const ownCompany =
                  resource.key === "companies" && row.code === companyCode;

                return (
                  <tr
                    key={rowId(resource, row)}
                    className={`transition-colors hover:bg-panel/35 ${
                      !row.isActive ? "opacity-75" : ""
                    }`}
                  >
                    {resource.columns.map((column) => (
                      <td key={column.key} className="px-3.5 py-3 align-top">
                        {renderCell(resource, row, column.key, {
                          ownUser,
                          ownCompany,
                          readOnly,
                          companyCode,
                        })}
                      </td>
                    ))}

                    <td className="px-3.5 py-3 align-top whitespace-nowrap">
                      <span
                        className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-semibold ${
                          row.isActive
                            ? "bg-status-success-bg text-status-success-text"
                            : "bg-panel text-text-muted border border-border"
                        }`}
                      >
                        <span
                          className={`h-1.5 w-1.5 rounded-full ${
                            row.isActive ? "bg-status-success-text" : "bg-text-muted"
                          }`}
                        />
                        {row.isActive ? "Aktif" : "Nonaktif"}
                      </span>
                    </td>

                    <td className="px-3.5 py-3 align-top">
                      <div className="flex flex-wrap justify-end gap-1.5">
                        {resource.fields.length > 0 &&
                          !protectedUser &&
                          !readOnly && (
                            <button
                              type="button"
                              className={rowActionButtonClass}
                              onClick={() => setAction({ kind: "edit", row })}
                            >
                              <Pencil size={13} />
                              Edit
                            </button>
                          )}

                        {!readOnly ? (
                          <>
                            <button
                              type="button"
                              className={rowActionButtonClass}
                              disabled={ownUser || ownCompany || protectedUser}
                              title={
                                ownUser || ownCompany
                                  ? "Akun atau perusahaan sendiri tidak dapat dinonaktifkan"
                                  : undefined
                              }
                              onClick={() => setAction({ kind: "status", row })}
                            >
                              <Power size={13} />
                              {row.isActive ? "Nonaktifkan" : "Aktifkan"}
                            </button>
                            <button
                              type="button"
                              className={rowDangerButtonClass}
                              disabled={ownUser || ownCompany || protectedUser}
                              title={
                                ownUser || ownCompany
                                  ? "Akun atau perusahaan sendiri tidak dapat dihapus"
                                  : `Hapus ${String(row.name ?? row.code ?? "")}`
                              }
                              onClick={() => setAction({ kind: "delete", row })}
                            >
                              <Trash2 size={13} />
                              Hapus
                            </button>
                          </>
                        ) : (
                          <span className="text-[11px] text-text-muted px-2 py-1">
                            Terkunci
                          </span>
                        )}

                        {resource.key === "users" && !protectedUser && (
                          <button
                            type="button"
                            className={rowActionButtonClass}
                            onClick={() => setAction({ kind: "password", row })}
                          >
                            <KeyRound size={13} />
                            Password
                          </button>
                        )}

                        {resource.key === "companies" && (
                          <button
                            type="button"
                            className={rowActionButtonClass}
                            onClick={() => setAction({ kind: "admin", row })}
                          >
                            <UserPlus size={13} />
                            Tambah Admin
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })}

              {rows.length === 0 && (
                <tr>
                  <td
                    colSpan={resource.columns.length + 2}
                    className="p-12 text-center"
                  >
                    <Inbox
                      size={32}
                      className="mx-auto text-text-muted mb-3 opacity-70"
                    />
                    <p className="font-semibold text-sm text-text">
                      {hasActiveFilter
                        ? "Data tidak ditemukan"
                        : `Belum ada ${resource.label.toLowerCase()}`}
                    </p>
                    <p className="text-xs text-text-muted mt-1 max-w-sm mx-auto">
                      {hasActiveFilter
                        ? "Coba ubah kata kunci pencarian atau atur ulang filter di atas."
                        : `Gunakan tombol Tambah ${resource.label} di kanan atas untuk mulai menambahkan data.`}
                    </p>
                    {hasActiveFilter && (
                      <button
                        type="button"
                        onClick={resetFilters}
                        className={`${buttonClass} mt-4`}
                      >
                        <RotateCcw size={13} />
                        Reset Filter
                      </button>
                    )}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* Pagination Footer */}
      <div className="flex flex-wrap justify-between items-center gap-3 text-xs text-text-muted pt-1">
        <span className="inline-flex items-center gap-1.5">
          <Database size={13} className="text-primary" />
          Menampilkan <strong className="text-text">{rows.length}</strong> dari{" "}
          <strong className="text-text">{total}</strong> data
          {query.isFetching && !query.isPending ? " · Memperbarui..." : ""}
        </span>
        {resource.paged && pages > 1 && (
          <div className="flex items-center gap-2">
            <button
              type="button"
              className={buttonClass}
              disabled={page <= 1 || query.isFetching}
              onClick={() => setPage(page - 1)}
            >
              Sebelumnya
            </button>
            <span className="px-2 font-medium text-text">
              Halaman {page} dari {pages}
            </span>
            <button
              type="button"
              className={buttonClass}
              disabled={page >= pages || query.isFetching}
              onClick={() => setPage(page + 1)}
            >
              Berikutnya
            </button>
          </div>
        )}
      </div>

      {action && (
        <MasterForm
          title={title}
          variant={action.kind === "delete" ? "danger" : "default"}
          description={
            description ??
            (resource.key === "users"
              ? "Perubahan jabatan atau role akan mengakhiri sesi aktif user. Unit kerja mengikuti jabatan yang dipilih."
              : undefined)
          }
          fields={fields}
          row={action.kind === "edit" ? action.row : undefined}
          company={
            action.kind === "admin" ? String(action.row?.code) : undefined
          }
          onClose={() => setAction(null)}
          onSubmit={save}
        />
      )}
    </section>
  );
}

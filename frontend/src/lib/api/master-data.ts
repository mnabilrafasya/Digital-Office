// Seluruh request melewati proxy sesi httpOnly Next.js (/api/v1).
export type MasterRow = Record<string, unknown>;

export interface MasterList {
  data: MasterRow[];
  meta?: { page: number; pageSize: number; total: number };
}

export interface Choice {
  value: string;
  label: string;
}

export interface CompanySettings {
  code: string;
  name: string;
  letterNumberFormat: string;
  numberPreview: string;
  watermarkText: string | null;
}

export async function masterRequest<T>(
  path: string,
  method = "GET",
  body?: unknown,
  companyCode?: string,
): Promise<T> {
  const headers: Record<string, string> = {};
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (companyCode) {
    headers["X-Company-Code"] = companyCode;
  }

  let requestUrl = `/api/v1${path}`;
  if (companyCode && !path.startsWith("/admin/companies")) {
    const sep = requestUrl.includes("?") ? "&" : "?";
    requestUrl = `${requestUrl}${sep}companyCode=${encodeURIComponent(companyCode)}`;
  }

  const response = await fetch(requestUrl, {
    method,
    cache: "no-store",
    credentials: "include",
    headers: Object.keys(headers).length ? headers : undefined,
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  if (response.status === 401) {
    window.location.assign(new URL("/login", window.location.origin).href);
    throw new Error("Sesi berakhir. Silakan masuk kembali.");
  }

  if (!response.ok) {
    const result = await response.json().catch(() => null);
    throw new Error(
      result?.error?.message ?? `Permintaan gagal (HTTP ${response.status}).`,
    );
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

export function valueAt(row: MasterRow, path: string): unknown {
  return path.split(".").reduce<unknown>(
    (value, key) =>
      value && typeof value === "object" ? (value as MasterRow)[key] : undefined,
    row,
  );
}

export async function allChoices(
  path: string,
  label: (row: MasterRow) => string,
  companyCode?: string,
): Promise<Choice[]> {
  const rows: MasterRow[] = [];
  let page = 1;

  while (true) {
    const separator = path.includes("?") ? "&" : "?";
    const result = await masterRequest<MasterList>(
      `${path}${separator}page=${page}&pageSize=100`,
      "GET",
      undefined,
      companyCode,
    );
    rows.push(...(result.data ?? []));
    if (!result.meta || rows.length >= result.meta.total || !result.data.length) {
      break;
    }
    page++;
  }

  return rows.map((row) => ({ value: String(row.id), label: label(row) }));
}

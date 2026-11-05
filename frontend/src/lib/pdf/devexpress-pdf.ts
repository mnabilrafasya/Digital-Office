// lib/pdf/devexpress-pdf.ts
// Pendukung ekspor PDF DevExpress RichEdit (exportToPdf). DevExpress membaca window.PDFDocument
// (pdfkit) dan window.BlobStream saat ekspor berjalan, jadi keduanya cukup dipasang tepat sebelum
// ekspor dan tidak membebani halaman yang tidak membuat PDF.

/** Penampung keluaran pdfkit (pengganti paket blob-stream): mengumpulkan potongan PDF menjadi Blob. */
class BlobStream {
  writable = true;
  private chunks: BlobPart[] = [];
  private handlers = new Map<string, Array<(...args: unknown[]) => void>>();

  on(event: string, fn: (...args: unknown[]) => void) {
    this.handlers.set(event, [...(this.handlers.get(event) ?? []), fn]);
    return this;
  }

  addListener(event: string, fn: (...args: unknown[]) => void) {
    return this.on(event, fn);
  }

  prependListener(event: string, fn: (...args: unknown[]) => void) {
    this.handlers.set(event, [fn, ...(this.handlers.get(event) ?? [])]);
    return this;
  }

  once(event: string, fn: (...args: unknown[]) => void) {
    const wrapper = (...args: unknown[]) => {
      this.removeListener(event, wrapper);
      fn(...args);
    };
    return this.on(event, wrapper);
  }

  removeListener(event: string, fn: (...args: unknown[]) => void) {
    this.handlers.set(event, (this.handlers.get(event) ?? []).filter((h) => h !== fn));
    return this;
  }

  off(event: string, fn: (...args: unknown[]) => void) {
    return this.removeListener(event, fn);
  }

  listeners(event: string) {
    return [...(this.handlers.get(event) ?? [])];
  }

  listenerCount(event: string) {
    return this.handlers.get(event)?.length ?? 0;
  }

  emit(event: string, ...args: unknown[]) {
    const list = this.handlers.get(event) ?? [];
    list.forEach((fn) => fn(...args));
    return list.length > 0;
  }

  write(chunk: Uint8Array) {
    this.chunks.push(new Uint8Array(chunk));
    return true;
  }

  end(chunk?: Uint8Array) {
    if (chunk) this.write(chunk);
    this.writable = false;
    this.emit("finish");
    this.emit("close");
    return this;
  }

  toBlob(type = "application/pdf") {
    return new Blob(this.chunks, { type });
  }

  toBlobURL(type = "application/pdf") {
    return URL.createObjectURL(this.toBlob(type));
  }
}

declare global {
  interface Window {
    PDFDocument?: unknown;
    BlobStream?: unknown;
  }
}

let loading: Promise<void> | null = null;

/** Memasang pdfkit dan BlobStream untuk ekspor PDF DevExpress (dimuat sekali). */
export function ensurePdfExportLibs(): Promise<void> {
  if (!loading) {
    loading = import("pdfkit/js/pdfkit.standalone")
      .then((mod) => {
        window.PDFDocument = (mod as { default?: unknown }).default ?? mod;
        window.BlobStream = BlobStream;
      })
      .catch((err) => {
        loading = null;
        throw err;
      });
  }
  return loading;
}

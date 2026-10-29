/**
 * Memastikan library PDFKit dan BlobStream telah siap di window browser
 * sebelum memanggil DevExpress RichEdit editor.downloadPdf().
 */
export async function ensurePdfKitLoaded(): Promise<boolean> {
  if (typeof window === "undefined") return false;

  const win = window as unknown as {
    PDFDocument?: unknown;
    BlobStream?: unknown;
    blobStream?: unknown;
  };

  // Jika kedua library sudah tersedia di window
  if (win.PDFDocument && (win.BlobStream || win.blobStream)) {
    win.BlobStream = win.BlobStream || win.blobStream;
    return true;
  }

  return new Promise((resolve) => {
    let pdfLoaded = Boolean(win.PDFDocument);
    let blobLoaded = Boolean(win.BlobStream || win.blobStream);

    const checkDone = () => {
      if (pdfLoaded && blobLoaded) {
        win.BlobStream = win.BlobStream || win.blobStream;
        resolve(true);
      }
    };

    if (pdfLoaded && blobLoaded) {
      checkDone();
      return;
    }

    if (!pdfLoaded) {
      const existingScript1 = document.querySelector('script[src*="pdfkit.standalone.js"]');
      if (existingScript1) {
        existingScript1.addEventListener("load", () => {
          pdfLoaded = true;
          checkDone();
        });
      } else {
        const s1 = document.createElement("script");
        s1.src = "/vendor/pdfkit.standalone.js";
        s1.async = true;
        s1.onload = () => {
          pdfLoaded = true;
          checkDone();
        };
        s1.onerror = () => {
          console.error("[PDFKitLoader] Gagal memuat /vendor/pdfkit.standalone.js");
          resolve(false);
        };
        document.head.appendChild(s1);
      }
    }

    if (!blobLoaded) {
      const existingScript2 = document.querySelector('script[src*="blob-stream.js"]');
      if (existingScript2) {
        existingScript2.addEventListener("load", () => {
          blobLoaded = true;
          win.BlobStream = win.BlobStream || win.blobStream;
          checkDone();
        });
      } else {
        const s2 = document.createElement("script");
        s2.src = "/vendor/blob-stream.js";
        s2.async = true;
        s2.onload = () => {
          blobLoaded = true;
          win.BlobStream = win.BlobStream || win.blobStream;
          checkDone();
        };
        s2.onerror = () => {
          console.error("[PDFKitLoader] Gagal memuat /vendor/blob-stream.js");
          resolve(false);
        };
        document.head.appendChild(s2);
      }
    }
  });
}


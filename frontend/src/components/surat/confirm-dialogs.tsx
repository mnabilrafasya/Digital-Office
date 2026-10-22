"use client";

// ─────────────────────────────────────────────
// components/surat/confirm-dialogs.tsx
// Dialog: Simpan Draft, Submit, Tinggalkan Halaman, Surat Terkirim
// ─────────────────────────────────────────────

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { StatusBadge } from "@/components/ui/status-badge";
import { CheckCircle2 } from "lucide-react";

export type DialogType = "simpan" | "submit" | "batal" | "sukses" | "draf_sukses" | null;

interface Props {
  dialog: DialogType;
  onClose: () => void;
  // Simpan
  isSaving: boolean;
  saveFailed: boolean;
  onSimpan: () => void;
  // Submit
  isSubmitting: boolean;
  submitFailed: boolean;
  submitError?: string | null;
  onSubmit: () => void;
  // Batal
  onLeave: () => void;
  // Sukses
  onNew: () => void;
  onGoToList: () => void;
}

export function ConfirmDialogs(p: Props) {
  const close = (open: boolean) => {
    if (!open && p.dialog !== "sukses" && p.dialog !== "draf_sukses") p.onClose();
  };

  return (
    <>
      {/* Dialog Simpan Draft */}
      <Dialog open={p.dialog === "simpan"} onOpenChange={close}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Simpan Draft Surat?</DialogTitle>
            <DialogDescription>
              Draft akan disimpan ke sistem. Anda dapat melanjutkan pengisian surat kapan saja dari Daftar Surat.
            </DialogDescription>
          </DialogHeader>
          {p.saveFailed && (
            <p role="alert" className="text-status-danger-text text-sm">Gagal menyimpan. Coba lagi.</p>
          )}
          <DialogFooter>
            <Button variant="outline" size="lg" onClick={p.onClose}>Batal</Button>
            <Button size="lg" onClick={p.onSimpan} disabled={p.isSaving}>
              {p.isSaving ? "Menyimpan..." : "Simpan"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Dialog Draft Berhasil Disimpan */}
      <Dialog open={p.dialog === "draf_sukses"} onOpenChange={close}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-emerald-700">
              <CheckCircle2 size={20} className="text-emerald-600 shrink-0" />
              Draft Berhasil Disimpan
            </DialogTitle>
            <DialogDescription>
              Draft surat Anda telah berhasil disimpan ke database. Anda dapat melihatnya di Daftar Surat atau melanjutkan pengisian.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="flex gap-2 sm:justify-end">
            <Button variant="outline" size="lg" onClick={p.onClose}>
              Lanjut Mengedit
            </Button>
            <Button size="lg" onClick={p.onGoToList}>
              Ke Daftar Surat
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Dialog Tinggalkan Halaman / Batal */}
      <Dialog open={p.dialog === "batal"} onOpenChange={close}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Batalkan Pembuatan Surat?</DialogTitle>
            <DialogDescription>
              Ada data yang telah diisi. Jika Anda keluar sekarang, perubahan yang belum disimpan akan hilang.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" size="lg" onClick={p.onClose}>Tetap di Form</Button>
            <Button variant="destructive" size="lg" onClick={p.onLeave}>Keluar ke Daftar Surat</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Dialog Submit */}
      <Dialog open={p.dialog === "submit"} onOpenChange={close}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Kirim Surat ke Reviewer?</DialogTitle>
            <DialogDescription>
              Surat akan dikirim ke <strong>Reviewer Tahap 1</strong> untuk ditinjau.
              Status surat akan berubah menjadi &ldquo;Menunggu Review&rdquo;.
            </DialogDescription>
          </DialogHeader>
          {p.submitFailed && (
            <p role="alert" className="text-status-danger-text text-sm font-medium">
              {p.submitError || "Gagal mengirim surat. Pastikan data surat sudah lengkap dan coba lagi."}
            </p>
          )}
          <DialogFooter>
            <Button variant="outline" size="lg" onClick={p.onClose}>Batal</Button>
            <Button size="lg" onClick={p.onSubmit} disabled={p.isSubmitting}>
              {p.isSubmitting ? "Mengirim..." : "Kirim ke Reviewer"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Dialog Surat Terkirim */}
      <Dialog open={p.dialog === "sukses"} onOpenChange={close}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Surat Terkirim</DialogTitle>
            <DialogDescription>
              Surat dikirim ke Reviewer Tahap 1. Status saat ini:{" "}
              <StatusBadge status="IN_REVIEW" className="align-middle" />
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" size="lg" onClick={p.onNew}>Buat Surat Lagi</Button>
            <Button size="lg" onClick={p.onGoToList}>Ke Daftar Surat</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

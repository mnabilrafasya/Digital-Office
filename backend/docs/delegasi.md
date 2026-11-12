# Halaman Manajemen Delegasi

Kebutuhan front-end untuk fitur delegasi. Aturan bisnis dan seluruh endpoint ada di [api.md](api.md#delegasi).

## Ringkas

Selama periode delegasi pegawai A ke pengganti B:
- tugas review/approval A hanya bisa dikerjakan B (atas nama A). A tidak mendapat tombol aksi;
- surat yang masuk ke inbox A juga masuk ke inbox B sebagai tembusan (CC);
- setelah periode selesai atau dicabut, semuanya kembali normal.

**Satu halaman "Manajemen Delegasi" untuk semua user.** Isi dan endpoint-nya mengikuti hak user dari `permissions` di `/auth/me`:

| | Pegawai | Admin (`USER_MANAGE`) |
|---|---|---|
| Yang dilihat | Jadwal sendiri + orang yang ia wakili | Semua jadwal di perusahaannya |
| Yang bisa diatur | Jadwal sendiri | Jadwal pegawai mana pun, termasuk dirinya |
| Field "Pegawai" di form | Tidak ada (otomatis dirinya) | Ada (wajib) |
| Endpoint | `/me/delegations` | `/admin/delegations` |

## Data delegasi

Satu jadwal (dipakai di semua respons):

```json
{
  "id": "12",
  "user": { "id": "3", "name": "Agus Setiawan", "positionName": "VP SDM" },
  "delegate": { "id": "17", "name": "Maya Anggraini", "positionName": "Staf Pengembangan SDM" },
  "from": "2026-11-12",
  "until": "2026-11-15",
  "note": "Dinas luar kota",
  "status": "ACTIVE",
  "createdBy": { "id": "13", "name": "Rudi Hartono", "positionName": "Administrator Sistem" },
  "createdAt": "2026-11-12T09:00:00+07:00",
  "updatedAt": "2026-11-12T09:00:00+07:00"
}
```

| Field | Arti |
|---|---|
| `user` | Pegawai yang mendelegasikan |
| `delegate` | Pengganti |
| `from`, `until` | Tanggal mulai dan selesai, `YYYY-MM-DD`, berlaku sehari penuh |
| `status` | `ACTIVE` (sedang berlaku), `UPCOMING` (belum mulai), `ENDED` (sudah lewat) |
| `createdBy` | Yang mengisi: pegawai itu sendiri atau admin |

Label status: `ACTIVE` = "Berlaku", `UPCOMING` = "Akan datang", `ENDED` = "Selesai".

## Isi halaman

**Bagian atas (semua user)**, dari `GET /me/delegations`:
- Kalau ada jadwal milik user yang `ACTIVE`, tampilkan banner: "Tugas Anda sedang didelegasikan ke {delegate.name} sampai {until}."
- Kartu **Saya mewakili** (`representing`): orang yang tugasnya sedang atau akan dikerjakan user. Hanya dibaca. Sembunyikan kalau kosong.

**Tabel delegasi**

| | Pegawai | Admin |
|---|---|---|
| Sumber | `GET /me/delegations` → `delegations` (berlaku dan akan datang) | `GET /admin/delegations?status=&userId=&page=&pageSize=` (berhalaman `data` + `meta`) |
| Filter | Tidak perlu | Tab status (Berlaku / Akan datang / Selesai / Semua) dan pencarian pegawai (`userId`, cocok sebagai pegawai maupun pengganti) |

| Kolom | Isi | Pegawai | Admin |
|---|---|---|---|
| Pegawai | `user.name`, `user.positionName` | – | ✓ |
| Pengganti | `delegate.name`, `delegate.positionName` | ✓ | ✓ |
| Periode | `from` s.d. `until` | ✓ | ✓ |
| Status | Badge `status` | ✓ | ✓ |
| Keterangan | `note` | ✓ | ✓ |
| Diisi oleh | `createdBy.name` (berguna bila diisi admin) | ✓ | ✓ |
| Aksi | Ubah, Hapus (hanya untuk `ACTIVE`/`UPCOMING`) | ✓ | ✓ |

## Form tambah/ubah

| Field | Input | Aturan |
|---|---|---|
| `userId` (admin saja) | Pencarian user: `GET /users/candidates?type=DELEGATE&q=` | Wajib saat tambah. Tidak bisa diganti saat ubah |
| `substituteId` | Pencarian user: `GET /users/candidates?type=DELEGATE&q=` | Wajib. Sembunyikan pegawai yang dipilih (atau diri sendiri) dari pilihan |
| `from` | Tanggal | Wajib |
| `until` | Tanggal | Wajib, tidak sebelum `from` dan tidak sebelum hari ini |
| `note` | Teks | Opsional, maksimal 200 karakter |

| Aksi | Pegawai | Admin | Hasil |
|---|---|---|---|
| Simpan baru | `POST /me/delegations` `{substituteId, from, until, note}` | `POST /admin/delegations` `{userId, substituteId, from, until, note}` | 201 + jadwal |
| Simpan ubahan | `PUT /me/delegations/{id}` `{substituteId, from, until, note}` | `PUT /admin/delegations/{id}` (body sama) | 200 + jadwal |
| Hapus / cabut | `DELETE /me/delegations/{id}` | `DELETE /admin/delegations/{id}` | 204 |

- Mengakhiri lebih awal = ubah `until` menjadi hari ini, atau hapus jadwalnya.
- Jadwal `ENDED` tidak bisa diubah atau dihapus, jadi sembunyikan tombolnya.
- Setelah simpan atau hapus, muat ulang tabel dan bagian atas.

## Pesan error

Tampilkan `error.message` apa adanya di dekat form.

| HTTP | Contoh pesan | Penyebab |
|---|---|---|
| 400 | "Pengganti wajib dipilih" | `substituteId` kosong |
| 400 | "Pengganti tidak boleh pegawai itu sendiri" | Pengganti sama dengan pegawai |
| 400 | "Tanggal mulai tidak boleh setelah tanggal selesai" | Urutan tanggal salah |
| 400 | "Tanggal selesai tidak boleh sebelum hari ini" | Periode sudah lewat |
| 400 | "Pegawai/Pengganti tidak ditemukan atau tidak aktif di perusahaan Anda" | User nonaktif atau beda perusahaan |
| 409 | "… sudah punya jadwal delegasi yang beririsan …" | Jadwal pegawai tumpang-tindih |
| 409 | "… sedang mendelegasikan tugasnya ke … jadi tidak bisa menjadi pengganti …" | Pengganti sendiri sedang berhalangan |
| 409 | "… sedang menjadi pengganti … supaya delegasi tidak berantai." | Pegawai sedang mewakili orang lain |
| 409 | "Delegasi yang sudah selesai tidak bisa diubah atau dihapus" | Jadwal `ENDED` |
| 404 | "Delegasi tidak ditemukan" | Pegawai mencoba mengubah jadwal orang lain |

## Perubahan di halaman lain

| Halaman | Field | Tampilan |
|---|---|---|
| Tugas Saya (`GET /letters?scope=tasks`) | `onBehalfOf` | Label "Mewakili {name}" pada baris tugas |
| Surat Saya dan Tugas Saya | `delegatedTo` | "Didelegasikan ke {delegate.name} s.d. {until}" di kolom penerima tugas |
| Detail surat | `onBehalfOf` | Banner "Anda mengerjakan surat ini atas nama {name}". Tombol tetap mengikuti `allowedActions` |
| Detail surat | `delegatedTo` | Info di samping penerima tugas saat ini. Untuk pemilik tugas, `allowedActions` kosong |
| Detail surat, persetujuan | `approval.onBehalfOf` | "Disetujui oleh {approvedBy.name} a.n. {onBehalfOf.name}" |
| Riwayat (`/history`) | `onBehalfOf` per aksi | "{actor.name} a.n. {onBehalfOf.name}" |
| Lacak (`/tracking`) | `position.delegatedTo`, `timeline[].onBehalfOf` | Label posisi dan linimasa sudah berisi kalimat jadi |
| Inbox | Tidak ada field baru | Tembusan untuk pengganti tampil sebagai item biasa dengan `source.recipientType = "CC"` |

Kalau pemilik tugas mencoba bertindak (misalnya lewat halaman yang belum dimuat ulang), respons 409 berisi pesan "Tugas Anda sedang didelegasikan ke … sampai …". Tampilkan pesan itu, lalu muat ulang detail surat.

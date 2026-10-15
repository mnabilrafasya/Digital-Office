package auth

import (
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// dummyCost sama dengan cost hash user (cmd/seed) supaya waktu perbandingannya setara
const dummyCost = 12

var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

// CheckPassword mengembalikan true bila password cocok dengan hash bcrypt
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// CheckDummyPassword menjalankan perbandingan bcrypt terhadap hash palsu dan selalu
// mengembalikan false. Dipakai saat user tidak ditemukan/tidak boleh login, supaya waktu
// respons tidak membocorkan NIK mana yang terdaftar.
func CheckDummyPassword(password string) bool {
	dummyHashOnce.Do(func() {
		// Hash dari string tetap; isinya tidak penting karena hasilnya selalu diabaikan
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("digioffice-dummy-password"), dummyCost)
	})
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
	return false
}

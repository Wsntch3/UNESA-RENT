package email

import "fmt"

func OTPSubject() string { return "Kode verifikasi UNESA Rent" }

func OTPBody(code string, minutes int) string {
	return fmt.Sprintf(
		"Kode verifikasi kamu: %s\n\nKode ini berlaku %d menit dan hanya bisa dipakai sekali.\n"+
			"Jangan bagikan kode ini kepada siapa pun, termasuk admin UNESA Rent.\n\n"+
			"Jika kamu tidak meminta kode ini, abaikan email ini.",
		code, minutes,
	)
}

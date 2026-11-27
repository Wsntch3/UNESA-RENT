package rental

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// latin1 mengganti karakter di luar Latin-1 karena core font fpdf tidak UTF-8.
func latin1(raw string) string {
	return strings.Map(func(r rune) rune {
		if r <= 255 {
			return r
		}
		return '?'
	}, raw)
}

func docNumber(prefix, id string) string {
	plain := strings.ReplaceAll(id, "-", "")
	if len(plain) > 8 {
		plain = plain[:8]
	}
	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().Year(), plain)
}

func newPDF(title string) *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(18, 18, 18)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 15)
	pdf.CellFormat(0, 9, latin1(title), "", 1, "C", false, 0, "")
	pdf.Ln(3)
	return pdf
}

func paragraph(pdf *fpdf.Fpdf, text string) {
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 5.5, latin1(text), "", "L", false)
	pdf.Ln(1)
}

func field(pdf *fpdf.Fpdf, label, value string) {
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(45, 6, latin1(label)+":", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, latin1(value), "", 1, "L", false, 0, "")
}

func money(value int) string {
	return fmt.Sprintf("Rp%d", value)
}

func programFromNIM(nim string) string {
	if len(nim) >= 5 && nim[2:5] == "120" {
		return "S1 Teknik Informatika"
	}
	return "Sesuai KTM"
}

func BuildAgreementPDF(r Rental, ownerNIM, renterNIM, location string) ([]byte, error) {
	pdf := newPDF("SURAT PERJANJIAN SEWA BARANG")
	paragraph(pdf, "Nomor: "+docNumber("SP", r.ID))

	paragraph(pdf, "Penyedia dan Penyewa menyepakati sewa barang dengan ketentuan berikut.")

	field(pdf, "Penyedia", r.Owner.Name)
	field(pdf, "NIM Penyedia", ownerNIM+" / "+programFromNIM(ownerNIM))
	field(pdf, "Penyewa", r.Renter.Name)
	field(pdf, "NIM Penyewa", renterNIM+" / "+programFromNIM(renterNIM))
	field(pdf, "Barang", r.Listing.Title)
	field(pdf, "Periode", r.StartDate+" s.d. "+r.EndDate)
	field(pdf, "Jumlah hari", fmt.Sprintf("%d hari", r.TotalDays))
	field(pdf, "Harga per hari", money(r.PricePerDaySnapshot))
	field(pdf, "Total pembayaran", money(r.TotalPrice))
	field(pdf, "Lokasi COD", location)

	pdf.Ln(2)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 6, "Ketentuan", "", 1, "L", false, 0, "")
	paragraph(pdf, "1. Penyewa memeriksa kondisi fisik barang saat serah terima dan memastikan kelengkapannya.")
	paragraph(pdf, "2. Penyedia menunjukkan KTM atau kartu identitas mahasiswa sebagai bukti identitas.")
	paragraph(pdf, "3. Barang wajib dikembalikan tepat pada tanggal yang disepakati dalam keadaan baik.")
	paragraph(pdf, "4. Kerusakan akibat pemakaian tidak wajar menjadi tanggung jawab penyewa.")
	paragraph(pdf, "5. Perjanjian ini adalah dokumen komunitas UNESA Rent, bukan akta notaris.")

	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(85, 6, "Penyedia,", "", 0, "L", false, 0, "")
	pdf.CellFormat(85, 6, "Penyewa,", "", 1, "L", false, 0, "")
	pdf.CellFormat(85, 6, "( "+r.Owner.Name+" )", "", 0, "L", false, 0, "")
	pdf.CellFormat(85, 6, "( "+r.Renter.Name+" )", "", 1, "L", false, 0, "")
	pdf.CellFormat(85, 6, "Dicatat: "+time.Now().Format("2006-01-02 15:04"), "", 0, "L", false, 0, "")
	pdf.CellFormat(85, 6, "Dicatat: "+time.Now().Format("2006-01-02 15:04"), "", 1, "L", false, 0, "")

	pdf.Ln(6)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.CellFormat(0, 5, "Dokumen dibuat otomatis oleh UNESA Rent.", "", 1, "C", false, 0, "")

	return toBytes(pdf), nil
}

func BuildReceiptPDF(r Rental, cod CODSummary) ([]byte, error) {
	pdf := newPDF("KWITANSI COD UNESA RENT")
	paragraph(pdf, "Nomor: "+docNumber("KW", r.ID))

	paid := 0
	if cod.AmountPaid != nil {
		paid = *cod.AmountPaid
	}
	due := 0
	if cod.AmountDue != nil {
		due = *cod.AmountDue
	}
	method := "cash"
	if cod.PaymentMethod != nil && *cod.PaymentMethod != "" {
		method = *cod.PaymentMethod
	}

	field(pdf, "Rental", r.ID)
	field(pdf, "Barang", r.Listing.Title)
	field(pdf, "Penyewa", r.Renter.Name)
	field(pdf, "Penyedia", r.Owner.Name)
	field(pdf, "Jumlah tagihan", money(due))
	field(pdf, "Jumlah dibayar", money(paid))
	field(pdf, "Metode", method)
	field(pdf, "Waktu", time.Now().Format("2006-01-02 15:04"))
	if cod.Notes != nil && *cod.Notes != "" {
		field(pdf, "Catatan", *cod.Notes)
	}

	pdf.Ln(4)
	paragraph(pdf, "Bukti pembayaran ini dicatat oleh sistem dan menjadi bagian dari arsip dokumen rental.")
	pdf.SetFont("Helvetica", "I", 8)
	pdf.CellFormat(0, 5, "Dokumen dibuat otomatis oleh UNESA Rent.", "", 1, "C", false, 0, "")
	return toBytes(pdf), nil
}

func toBytes(pdf *fpdf.Fpdf) []byte {
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil
	}
	return buf.Bytes()
}

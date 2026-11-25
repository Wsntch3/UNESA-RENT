package catalog

import (
	"errors"
	"testing"
	"time"
)

const validUUID = "11111111-1111-4111-8111-111111111111"

func TestValidateListingInput(t *testing.T) {
	valid := ListingInput{
		Title:            "Kamera Canon",
		Description:      "Kamera mulus.",
		CategoryID:       validUUID,
		CampusLocationID: validUUID,
		PricePerDay:      50000,
		ConditionPercent: 90,
	}

	got, err := validateListingInput(valid)
	if err != nil {
		t.Fatalf("input valid ditolak: %v", err)
	}
	if got.Title != "Kamera Canon" {
		t.Errorf("judul tidak di-trim: %q", got.Title)
	}

	cases := []struct {
		name   string
		mutate func(*ListingInput)
		field  string
	}{
		{"judul pendek", func(in *ListingInput) { in.Title = "ab" }, "title"},
		{"harga nol", func(in *ListingInput) { in.PricePerDay = 0 }, "price_per_day"},
		{"harga terlalu besar", func(in *ListingInput) { in.PricePerDay = 10_000_001 }, "price_per_day"},
		{"kondisi di luar rentang", func(in *ListingInput) { in.ConditionPercent = 101 }, "condition_percent"},
		{"kategori bukan uuid", func(in *ListingInput) { in.CategoryID = "bukan-uuid" }, "category_id"},
		{"lokasi kosong", func(in *ListingInput) { in.CampusLocationID = "" }, "campus_location_id"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := valid
			tc.mutate(&in)
			_, err := validateListingInput(in)

			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("mau ValidationError, dapat %v", err)
			}
			if _, ok := ve.Fields[tc.field]; !ok {
				t.Errorf("field %q tidak ada di %v", tc.field, ve.Fields)
			}
		})
	}
}

func TestParseSearch(t *testing.T) {
	filter, err := parseSearch(SearchParams{
		Query:    "  kamera ",
		Category: "elektronik",
		Location: validUUID,
		MinPrice: "10000",
		MaxPrice: "20000",
		Sort:     "price_asc",
		Limit:    500,
		Offset:   -3,
	})
	if err != nil {
		t.Fatalf("search valid ditolak: %v", err)
	}
	if filter.Query != "kamera" {
		t.Errorf("query tidak di-trim: %q", filter.Query)
	}
	if filter.Limit != MaxListLimit {
		t.Errorf("limit harus dibatasi %d, dapat %d", MaxListLimit, filter.Limit)
	}
	if filter.Offset != 0 {
		t.Errorf("offset negatif harus jadi 0, dapat %d", filter.Offset)
	}

	cases := []struct {
		name   string
		params SearchParams
		field  string
	}{
		{"lokasi bukan uuid", SearchParams{Location: "abc"}, "location"},
		{"harga bukan angka", SearchParams{MinPrice: "murah"}, "min_price"},
		{"min lebih besar dari max", SearchParams{MinPrice: "20000", MaxPrice: "10000"}, "max_price"},
		{"sort tidak dikenal", SearchParams{Sort: "termurah"}, "sort"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseSearch(tc.params)

			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("mau ValidationError, dapat %v", err)
			}
			if _, ok := ve.Fields[tc.field]; !ok {
				t.Errorf("field %q tidak ada di %v", tc.field, ve.Fields)
			}
		})
	}
}

func TestParseBlockRange(t *testing.T) {
	today := time.Date(2026, 11, 24, 0, 0, 0, 0, jakarta)

	start, end, err := parseBlockRange(BlockInput{
		StartDate: "2026-11-25",
		EndDate:   "2026-11-27",
	}, today)
	if err != nil {
		t.Fatalf("periode valid ditolak: %v", err)
	}
	if start.Format(DateFormat) != "2026-11-25" || end.Format(DateFormat) != "2026-11-27" {
		t.Errorf("periode salah: %s - %s", start, end)
	}

	cases := []struct {
		name  string
		in    BlockInput
		field string
	}{
		{"selesai sebelum mulai", BlockInput{StartDate: "2026-11-27", EndDate: "2026-11-25"}, "end_date"},
		{"mulai di masa lalu", BlockInput{StartDate: "2026-11-23", EndDate: "2026-11-25"}, "start_date"},
		{"format salah", BlockInput{StartDate: "25-11-2026", EndDate: "2026-11-27"}, "start_date"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := parseBlockRange(tc.in, today)

			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("mau ValidationError, dapat %v", err)
			}
			if _, ok := ve.Fields[tc.field]; !ok {
				t.Errorf("field %q tidak ada di %v", tc.field, ve.Fields)
			}
		})
	}
}

func TestUpcomingPeriodsDropsPast(t *testing.T) {
	periods := upcomingPeriods([]Block{
		{StartDate: "2020-01-01", EndDate: "2020-01-05"},
		{StartDate: "2999-01-01", EndDate: "2999-01-03"},
	})

	if len(periods) != 1 || periods[0].StartDate != "2999-01-01" {
		t.Fatalf("periode masa lalu harus dibuang, dapat %+v", periods)
	}
}

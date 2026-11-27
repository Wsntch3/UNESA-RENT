# UNESA Rent — Interactive Prototype & Specification Matrix (Phase 6B)
**Platform**: Web Application (Desktop 1440px / Mobile 390px responsive)  
**Target User**: Mahasiswa Fakultas Teknik Universitas Negeri Surabaya (FT UNESA)  
**Model**: Peer-to-Peer (P2P) Rental — Single Unified Account (Renter + Owner) + Dedicated Admin Workspace  
**Design System**: `unesa-rent-playful-geometric-marketplace` (DESIGN.md / Cobalt #2A44E8 / Warm Ivory #FFFAF5)

---

## 1. Master Screen Inventory & Route Mapping (20 Screen Assets)

| ID | Title / Role | Archetype | Shell / Nav Model | Primary Flow |
|---|---|---|---|---|
| **01** | `Jelajahi Barang — UNESA Rent (Refined)` | Marketplace Catalog | Global Student Top Nav + Footer | Flow A (Start), Auth Gate |
| **02** | `Detail Barang — Sanwa CD800a Multimeter (Refined)` | Product Specification | Global Student Top Nav + Footer | Flow A |
| **03** | `Pengajuan Sewa & Penawaran — Sanwa CD800a (Updated)` | Rental Form & Offer | Global Student Top Nav + Footer | Flow A |
| **04** | `Pesan & Negosiasi Sewa — UNESA Rent (Refined)` | Chat & Dual-Negotiation | Global Student Top Nav + Footer | Flow A & Flow B |
| **05** | `Tambah Barang — Sewakan Barangmu` | Listing Creation Form | Global Student Top Nav + Footer | Flow B |
| **06** | `Kelola Barang Saya — UNESA Rent` | Owner Inventory Manager | Global Student Top Nav + Footer | Flow B |
| **07** | `Dashboard Penyewaan — UNESA Rent` | Owner Analytics & Activity | Global Student Top Nav + Footer | Flow B |
| **08** | `Detail Pengajuan Sewa — UNESA Rent` | Incoming Offer Review | Global Student Top Nav + Footer | Flow B |
| **09** | `Login — UNESA Rent` | Student Auth | Blank / Centered Split Auth | Auth Entry |
| **10** | `Registrasi Mahasiswa — UNESA Rent` | Student Registration | Blank / Centered Split Auth | Auth Flow |
| **11** | `Verifikasi Identitas Mahasiswa — UNESA Rent` | KTM Upload & Consent | Global Student Top Nav + Footer | Verification Gate |
| **12** | `Profil & Status Verifikasi — UNESA Rent` | Student Account Profile | Global Student Top Nav + Footer | Hub / Role Switcher |
| **13** | `Penyewaan Saya — UNESA Rent` | Active Rentals & History | Global Student Top Nav + Footer | Flow A & Flow B Hub |
| **14** | `Detail Penyewaan — UNESA Rent` | Agreement & Status Timeline | Global Student Top Nav + Footer | Flow A & Flow B |
| **15** | `Serah Terima Barang & Pencatatan COD — UNESA Rent` | Physical Handover & COD Doc | Global Student Top Nav + Footer | Flow A & Flow B Handover |
| **16** | `Pengembalian Barang — UNESA Rent` | Return Inspection & Close | Global Student Top Nav + Footer | Flow A & Flow B Return |
| **17** | `Dashboard Admin — UNESA Rent` | BEM FT Admin Dashboard | Dedicated Admin Left Sidebar + Topbar | Flow C |
| **18** | `Verifikasi Mahasiswa — UNESA Rent` | KTM Queue & Masked Review | Dedicated Admin Left Sidebar + Topbar | Flow C |
| **19** | `Moderasi Pengguna & Listing — UNESA Rent` | User/Listing Governance | Dedicated Admin Left Sidebar + Topbar | Flow C |
| **20** | `Laporan Permasalahan — UNESA Rent` | Dispute & Issue Dossier | Dedicated Admin Left Sidebar + Topbar | Flow C |

---

## 2. Interactive User Journeys & State Transitions

### Flow A: The Student Renter Journey
1. **Discovery (Screen 01)**: User searches *"Multimeter"* → Filter by FT department (*Teknik Elektro*) → Clicks card **Multimeter Digital Sanwa CD800a**.
2. **Product Inspection (Screen 02)**: Reviews calibrated specs, owner badge (*✓ KTM Terverifikasi*), rental terms (Rp25.000/hari, COD Gedung E1) → Clicks **"Ajukan Sewa Sekarang"**.
3. **Offer Submission (Screen 03)**: Selects rental dates (14–17 Nov = 3 days). Standard price Rp75.000. Uses counteroffer field to propose **Rp65.000** for academic lab module → Clicks **"Kirim Pengajuan"**.
   - *State Transition*: Proposal becomes `Menunggu Tinjauan Pemilik`.
4. **Negotiation (Screen 04)**: Enters conversation thread with owner (*Rian Hidayat*). Both discuss pickup at Gazebo Gedung E1. Owner sends agreed counteroffer card Rp65.000. Renter clicks **"Setujui Penawaran"**.
   - *State Transition*: Proposal locks into `Disepakati`.
5. **Rental Hub (Screen 13)**: Renter navigates to `Penyewaan Saya` (view: *Sebagai Penyewa*) → sees card with status `Sedang Disewa (Hari 2 dari 3)` or `Menunggu Serah Terima` → Clicks **"Lihat Detail"** / **"Mulai Serah Terima"**.
6. **Timeline & Single Source of Truth (Screen 14)**: Inspects contractual digital receipt #RENT-8821 with 5-stage vertical timeline.
7. **Physical Handover & COD (Screen 15)**: Meets at Gazebo E1 → Inspects physical KTM without leaving it as deposit → Inspects probe cables and calibration seal → Clicks **"Uang Tunai Telah Diserahkan (Rp65.000)"** → Clicks **"Konfirmasi Barang Diterima"**.
   - *State Transition*: `Menunggu Serah Terima` → `Sedang Disewa (Aktif)`.
8. **Return & Completion (Screen 16)**: Practical session concludes → Renter uploads 2 return photos → Side-by-side comparison confirms intact status → Owner clicks **"Konfirmasi Barang Diterima & Selesaikan Sewa"**.
   - *State Transition*: `Sedang Disewa` → `Selesai` (Archived to Riwayat).

---

### Flow B: The Student Item Owner Journey
1. **Creation (Screen 05)**: Owner navigates to `+ Tambah Barang` → Enters title, technical specs, daily rental rate, required accessories, and selects designated campus COD spot (*Gazebo E1 / Lobi Gedung A9*) → Clicks **"Tayangkan Listing"**.
2. **Inventory Management (Screen 06)**: Views active items. Rented items are protected from deletion or edits while active.
3. **Incoming Request Review (Screen 08)**: Receives offer from *Dimas Arya* (NIM 220512040***, S1 Elektro). Reviews proposed dates and price → Clicks **"Buka Diskusi & Negosiasi"** or **"Terima Penawaran"**.
4. **Agreement Confirmation (Screen 04)**: Confirms mutually agreed terms. Both students lock agreement.
5. **Handover Execution (Screen 15)**: Hands over Sanwa CD800a at Gazebo E1 → Clicks **"Konfirmasi Terima Rp65.000"** → Clicks **"Konfirmasi Barang Diserahkan"**.
6. **Return Verification (Screen 16)**: Inspects returned item condition side-by-side with original photos. Checks off the 4-point verification checklist → Closes transaction with **"Selesaikan Sewa"** (or triggers *Laporkan Kendala ke BEM FT* if damaged).

---

### Flow C: The Administrator Workspace Journey (BEM FT UNESA)
1. **Admin Portal (Screen 17)**: Access via `Portal Administrasi Kampus FT` → Dashboard shows 348 Verified Students, 128 Active Listings, 7 Pending KTMs, 2 Active Dispute Reports.
2. **KTM Verification Audit (Screen 18)**:
   - Selects entry in queue (*Dimas Arya Pradana*, S1 Elektro '22).
   - Examines masked digital card image, checks 4-point verification criteria (SSO email match, valid academic year, unmanipulated image).
   - Executes **"Setujui Verifikasi Identitas"** (Action recorded in system log with reviewer ID).
3. **Community & Listing Moderation (Screen 19)**:
   - Evaluates flagged listing or user compliance scores (100% COD compliance vs warning notices).
   - Controls: *Kirim Peringatan Sistem*, *Reset Pengajuan KTM*, *Tangguhkan Akun*.
4. **Dispute & Report Resolution (Screen 20)**:
   - Opens Dossier #RPT-2026-104 (*Keterlambatan Pengembalian Kamera DSLR Canon EOS 200D II*).
   - Reviews transaction #RENT-8819 audit trail, initial handover photos, and chat screenshots.
   - Logs mediator notes, sets status to `Sedang Ditinjau` / `Panggilan Mediasi Kampus`.

---

## 3. Auth & Access Gating Rules
- **Public Unverified Access**: Browsing marketplace catalog (Screen 01), searching items, and viewing product specs (Screen 02) is open.
- **Verification Gated Actions**: Clicking *Ajukan Sewa*, *Tambah Barang*, or *Kirim Penawaran* checks verification state.
  - If `Belum Verifikasi` or `Menunggu Verifikasi`: Displays modal or redirects to **Screen 11 (Verifikasi Identitas Mahasiswa)**.
  - If `Terverifikasi`: Permitted to execute transactions immediately.

---

## 4. Production Handoff & Technical Validation Matrix

| Aspect | Implementation Guidelines & Token Mapping |
|---|---|
| **Color Tokens** | Primary Cobalt `#2A44E8`, Page Canvas `#FFFAF5`, Card Surfaces `#FFFFFF`, Border Ink `#2E2E2E` (10% alpha), Accent Yellow `#FED230`. |
| **Typography** | Font Family: `Inter, sans-serif`. Heading scales: 32px (Hero/Page), 20px (Section), 14px (Body), 12px (Caption/Badge). |
| **Component Spacing** | Strict 8px rhythmic grid (8px, 16px, 24px, 32px, 48px). Card corner radius: 12px to 16px. |
| **Known Prototype Limitations** | (1) Camera & file uploads are simulated with fictional student assets; (2) In-person COD payment confirmation is a mutual acknowledgment log, not an electronic banking gateway; (3) Chat operates via simulated socket events. |

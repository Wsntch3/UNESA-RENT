# Frontend UNESARENT

Antarmuka React + Vite untuk marketplace sewa barang UNESA. State auth, pemanggilan API, guard halaman, dan tampilan sudah ada di satu folder ini.

## Menjalankan

```bash
npm install
npm run dev
```

Buka `http://localhost:5173`. Untuk development, proxy ke backend diatur di `vite.config.js` lewat `VITE_PROXY_TARGET`.

Untuk produksi:

```bash
npm run build
```

Output ada di `dist/`.

## Env

```text
VITE_API_URL=https://<domain-backend>
```

Nilai ini dipakai saat build. Kalau diubah, build ulang dan deploy ulang. CSP juga dibangkitkan dari nilai ini, jadi jangan mengubahnya setelah build tanpa build ulang.

## Alur auth

- Mahasiswa daftar dengan email `@mhs.unesa.ac.id`, minta OTP, verifikasi, lalu login.
- Admin login dengan password. Kalau `ADMIN_LOGIN_OTP=true`, muncul form "Verifikasi Admin"; token baru dibuat setelah kode 6 digit benar.
- Token disimpan di localStorage. `AuthProvider` memulihkan sesi lewat `/api/auth/me`.
- `RequireAuth` dan `RequireRole` menjaga halaman privat.

## Struktur penting

```text
src/auth/     AuthContext, useAuth, tokenStorage, RequireAuth
src/lib/      http.js, authApi.js, api.js, format.js
src/pages/    halaman publik, dashboard, admin
src/router.jsx router hash
src/index.css seluruh style
```

Error API dilempar sebagai `ApiError` dengan `status`, `code`, dan `fields`. Pesan yang ditampilkan ke user sudah dalam Bahasa Indonesia.

## Deploy

Vercel:

```text
Root Directory: frontend
Framework: Vite
Build: npm run build
Output: dist
```

Routing memakai hash (`/#/jelajahi`), jadi tidak perlu rewrite SPA.

# timesheet-agent

Menyusun timesheet bulanan dari riwayat commit **GitHub + GitLab**, dengan format
yang sama persis dengan Google Sheet timesheet vendor:

```
DATE | Task / Activity/ Project Name / Ticket no. | START TIME | END TIME | REMARK
```

Output default berupa **TSV** — copy lalu paste ke sel pertama di Google Sheet,
otomatis terbagi ke 5 kolom. Tanpa OAuth / Sheets API.

## Struktur

```
timesheet-agent/
├── cmd/timesheet/main.go          # entrypoint: load config, tarik semua sumber
├── internal/config/config.go      # baca config.json
├── internal/vcs/commit.go         # tipe commit netral + ekstraksi ticket/noise
├── internal/github/client.go      # klien GitHub (repo private)
├── internal/gitlab/client.go      # klien GitLab self-hosted
├── internal/timesheet/build.go    # logika commit -> baris timesheet
├── internal/timesheet/render.go   # render TSV / markdown
├── config.example.json
├── Makefile · .env.example
```

## Setup

Butuh Go 1.21+, GitHub PAT (scope baca repo), dan GitLab PAT (scope `read_api`).

```bash
cp .env.example .env                       # isi GITHUB_TOKEN & GITLAB_TOKEN
cp config.example.json config.json         # sudah berisi daftar repo kamu
# edit config.json: isi author.emails dengan email git kamu
export $(grep -v '^#' .env | xargs)
```

## Pakai

Untuk periode timesheet 16 Agustus – 15 September 2026:

```bash
# cek dulu sebagai tabel
go run ./cmd/timesheet -since 2026-08-16 -until 2026-09-15 -format md

# kalau sudah pas, generate TSV untuk dipaste ke Sheet
go run ./cmd/timesheet -since 2026-08-16 -until 2026-09-15 > timesheet.tsv
```

Bisa juga `-month 2026-08` untuk satu bulan penuh.

### Flag

| Flag       | Default       | Keterangan                              |
|------------|---------------|-----------------------------------------|
| `-config`  | `config.json` | path file config                        |
| `-since` / `-until` | —    | rentang `YYYY-MM-DD` (untuk 16–15)       |
| `-month`   | —             | alternatif: `YYYY-MM` (bulan penuh)     |
| `-format`  | `tsv`         | `tsv` atau `md`                         |
| `-standup` | `true`        | baris Daily Stand Up tiap hari kerja    |
| `-qa`      | `true`        | baris Stand By QA Testing tiap hari kerja |

## Config

`config.json` menentukan sumber & identitas author:

- **GitHub** difilter server-side pakai `author.github_login`.
- **GitLab** tidak bisa filter author di server, jadi commit dicocokkan lokal
  dengan `author.names` / `author.emails` — pastikan email git kamu benar.
- `gitlab.insecure_tls: true` hanya kalau instance internal pakai CA sendiri.

## Aturan mapping

- Hari kerja Senin–Jumat; akhir pekan dilewati.
- Jam default `08.00–17.00` di baris pertama tiap tanggal.
- Tiap hari kerja: `Daily Stand Up` → baris per commit → `Stand By QA Testing`.
- Commit sampah (`merge`, `wip`, `revert`, `bump version`) diabaikan.
- Ticket `NC-/CUS-/US-xxx` diekstrak otomatis; commit lintas repo digabung per hari.
- `Working on` di hari pertama ticket, `Continue Working on` untuk lanjutan.
- `In Progress` jika ticket masih muncul di hari berikutnya, `Done` di hari terakhir.

## Yang TIDAK otomatis (lengkapi manual saat review)

Meeting (sprint planning, grooming), cuti, dan standby di hari tanpa commit —
hal-hal ini tidak meninggalkan jejak di Git. Selalu cek sebelum submit.

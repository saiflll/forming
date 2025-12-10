# Forming App - Production Monitor

Dashboard monitoring produksi MDCW dengan fitur:

## ✨ Fitur Utama

### 📊 Dashboard Real-time

- Summary produksi 1 jam terakhir per line
- Data log dengan filter status (OK/Under/Over)
- Sorting by weight atau newest
- Auto-refresh data

### 📅 **Date Range Filter** (NEW!)

- Filter data berdasarkan tanggal custom
- Pilih range: dari tanggal - sampai tanggal
- Support filter kombinasi dengan status & sorting

### 📤 **Export ke Google Spreadsheet** (NEW!)

- Auto-export setiap data masuk ke Google Sheets
- Real-time sync tanpa delay
- Async & non-blocking
- Lihat setup: [GOOGLE-SHEETS-SETUP.md](./GOOGLE-SHEETS-SETUP.md)

### 💾 Data Export

- Export to CSV untuk analisa offline
- Skip log untuk data dengan weight = 0

### 🎯 Fitur Lainnya

- Tab dinamis per prefix/line
- MQTT subscription real-time
- PostgreSQL database storage
- Modern & responsive UI

## 🚀 Quick Start

### 1. Setup Database & MQTT

```bash
# Edit kredensial di .env.local
cp env.local.template .env.local
nano .env.local
```

### 2. (Optional) Setup Google Sheets

Ikuti panduan lengkap: [GOOGLE-SHEETS-SETUP.md](./GOOGLE-SHEETS-SETUP.md)

### 3. Install Dependencies

```bash
go mod download
```

### 4. Jalankan Aplikasi

**Dengan SSH Tunnel (Recommended):**

```bash
start-with-tunnel.bat
```

**Atau manual:**

```bash
go run .
```

Buka browser: http://localhost:3000

## 📋 Environment Variables

```bash
# Database
DB_HOST=172.20.100.11
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=servfi

# MQTT
MQTT_HOST=172.20.100.11
MQTT_PORT=1883
MQTT_USER=your_user
MQTT_PASSWORD=your_password

# Google Sheets (Optional)
GOOGLE_SHEETS_CREDENTIALS=base64_encoded_credentials
GOOGLE_SPREADSHEET_ID=your_spreadsheet_id
GOOGLE_SHEET_NAME=Production Data
```

## 🎨 UI Features

### Date Range Picker

```
[📅 Dari Tanggal] [📅 Sampai Tanggal] [Clear Filter]
```

- Pilih custom date range
- Auto-load data saat tanggal berubah
- Clear filter untuk kembali ke mode real-time

### Filter Options

- **Status**: Semua / OK / Under / Over
- **Sort**: Terbaru / Berat Tertinggi / Berat Terendah
- **Prefix Tab**: Semua Data / Line-specific

## 📊 Data Flow

```
IoT Device (MQTT)
    ↓
MQTT Broker
    ↓
Forming App (subscribe)
    ↓
    ├──→ PostgreSQL (insert)
    ├──→ Google Sheets (async export) [NEW!]
    └──→ Web Dashboard (HTMX)
```

## 🔧 Tech Stack

- **Backend**: Go + Fiber
- **Frontend**: HTMX + Alpine.js
- **Database**: PostgreSQL
- **Message Queue**: MQTT
- **Cloud Integration**: Google Sheets API
- **UI**: Vanilla CSS (responsive)

## 📁 File Structure

```
forming/
├── main.go                    # Main application
├── sheets.go                  # Google Sheets integration [NEW]
├── date_filter.go             # Date range filtering [NEW]
├── skip_log.go               # Skip log handler
├── views/
│   ├── index.html            # Dashboard with date picker [UPDATED]
│   ├── data_list.html        # Data table template
│   └── summary.html          # Summary cards
├── GOOGLE-SHEETS-SETUP.md    # Setup guide [NEW]
└── README-TUNNEL.md          # SSH tunneling guide
```

## 🆕 What's New

### v2.0 - Date Range & Sheets Integration

**✅ Date Range Filter**

- Custom date picker untuk filter data
- Support start_date & end_date parameters
- Combine dengan filter lain (status, sort, prefix)

**✅ Google Sheets Export**

- Auto-export ke spreadsheet
- Async & non-blocking
- Configurable via environment variables
- Complete setup documentation

**✅ UI Improvements**

- Date picker controls
- Clear filter button
- Better responsive design

## 📝 API Endpoints

### Data Endpoints

```
GET /data-list?status=all&sort=newest
GET /data-by-prefix?prefix=LINE-1&status=1
GET /data-by-date?start_date=2025-12-01&end_date=2025-12-10  [NEW]
```

### Summary & Metadata

```
GET /summary              # Produksi 1 jam terakhir
GET /prefixes             # List semua line/prefix [NEW]
GET /skip-log             # Log data yang di-skip
```

## 🔐 Security Notes

⚠️ **PENTING:**

- Jangan commit `.env.local` ke Git
- Jangan commit `*credentials*.json` ke Git
- Gunakan SSH tunnel untuk koneksi ke server production
- Rotate service account keys secara berkala

## 📖 Documentation

- [Google Sheets Setup](./GOOGLE-SHEETS-SETUP.md) - Setup integrasi Google Sheets
- [SSH Tunnel Guide](./README-TUNNEL.md) - Setup SSH tunneling
- [Deployment Guide](./DEPLOYMENT.md) - Deploy ke production

## 🐛 Troubleshooting

### Google Sheets tidak sync

1. Check environment variables sudah set
2. Verify spreadsheet sudah di-share dengan service account
3. Check log untuk error message
4. Lihat [GOOGLE-SHEETS-SETUP.md](./GOOGLE-SHEETS-SETUP.md)

### Date filter tidak jalan

1. Pastikan format date: YYYY-MM-DD
2. Check browser console untuk JavaScript errors
3. Verify `/data-by-date` endpoint response

## 📞 Support

Jika ada masalah, check:

1. Log aplikasi untuk error messages
2. Browser console untuk JS errors
3. Database connection status
4. MQTT broker connectivity

---

**Made with ❤️ for Production Monitoring**

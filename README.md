# 🚀 FORMING APP - Production Monitor

## 📦 Production Deployment (Docker)

### **Quick Deploy**

```bash
# 1. Clone atau upload aplikasi ke server
git clone <repository-url> forming
cd forming

# 2. Buat file .env dari template
cp env.local.template .env.local
nano .env.local

# 3. Deploy dengan Docker Compose
docker-compose up -d

# 4. Monitor logs
docker logs forming-app -f
```

### **Access Dashboard**

```
http://<server-ip>:3000
```

---

## 🔧 Configuration

### **File `.env.local`**

```env
DB_HOST=postgres_db
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password_rahasia_anda
DB_NAME=servfi
MQTT_HOST=emqx
MQTT_PORT=1883
MQTT_USER=
MQTT_PASSWORD=
TZ=Asia/Jakarta
```

**Catatan:**

- Untuk Docker: gunakan service names (`postgres_db`, `emqx`)
- Untuk local development: gunakan `localhost`

---

## 📊 MQTT Message Format

**Topic:** `production/mdcw`

**Payload:**

```json
{
  "ts": "2025-12-09 10:00:00",
  "reg2": 1250,
  "reg5": 1,
  "reg114": 500,
  "prefix": "mdcw2"
}
```

**Field Description:**

| Field    | Type   | Description                         | Example               |
| -------- | ------ | ----------------------------------- | --------------------- |
| `ts`     | string | Timestamp (YYYY-MM-DD HH:MM:SS)     | "2025-12-09 10:00:00" |
| `reg2`   | int    | Total Pack Count                    | 1250                  |
| `reg5`   | int    | Status Code (1=OK, 2=Under, 3=Over) | 1                     |
| `reg114` | int    | Weight in grams                     | 500                   |
| `prefix` | string | Machine/Node identifier             | "mdcw2"               |

**Status Codes:**

- `1` = ✓ OK (Normal weight)
- `2` = ⚠ Under (Underweight)
- `3` = ✗ Over (Overweight)

---

## 🐳 Docker Commands

### **Start Application**

```bash
docker-compose up -d
```

### **Stop Application**

```bash
docker-compose down
```

### **Rebuild Application**

```bash
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### **View Logs**

```bash
# Live logs
docker logs forming-app -f

# Last 100 lines
docker logs forming-app --tail 100
```

### **Restart Application**

```bash
docker-compose restart
```

---

## 🎨 Dashboard Features

- ✅ **Multi-prefix support** - Tab dinamis per mesin/node
- ✅ **Real-time updates** - Auto-refresh setiap 60 detik
- ✅ **Search & Filter** - Filter by status, sort by weight/time
- ✅ **Data Export** - Export ke CSV untuk audit
- ✅ **Summary Stats** - Total production last 1 hour per prefix
- ✅ **Skip Log** - Track data dengan weight = 0
- ✅ **Status Badges** - Visual indicators (OK/Under/Over)

---

## 📂 Project Structure

```
forming/
├── main.go              # Main application
├── skip_log.go          # Skip log handler
├── go.mod               # Go dependencies
├── go.sum               # Go checksums
├── Dockerfile           # Docker build config
├── docker-compose.yml   # Docker services config
├── .env.local           # Environment variables (gitignored)
├── env.local.template   # Template untuk .env.local
├── .gitignore           # Git ignore rules
├── README.md            # This file
├── views/               # HTML templates
│   ├── index.html       # Main dashboard
│   ├── data_list.html   # Data list partial
│   └── summary.html     # Summary partial
└── public/              # Static assets
    └── js/
        ├── htmx.min.js
        └── alpine.min.js
```

---

## 🔧 Database

### **Tables**

**1. production_mdcw**

Stores all valid production data:

```sql
CREATE TABLE production_mdcw (
    id SERIAL PRIMARY KEY,
    ts VARCHAR(50),
    reg2 INTEGER,
    reg5 INTEGER,
    reg114 INTEGER,
    prefix VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**2. skip_log**

Stores skipped data (weight = 0):

```sql
CREATE TABLE skip_log (
    id SERIAL PRIMARY KEY,
    ts VARCHAR(50),
    reg2 INTEGER,
    reg5 INTEGER,
    reg114 INTEGER,
    prefix VARCHAR(50),
    reason VARCHAR(100),
    skipped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

Tables are **automatically created** on first run.

---

## 🔍 Troubleshooting

### **Cannot connect to database**

1. Check if PostgreSQL container is running:

   ```bash
   docker ps | grep postgres
   ```

2. Check database connection from app:
   ```bash
   docker logs forming-app | grep -i postgres
   ```

### **Cannot connect to MQTT**

1. Check if EMQX container is running:

   ```bash
   docker ps | grep emqx
   ```

2. Check MQTT connection from app:
   ```bash
   docker logs forming-app | grep -i mqtt
   ```

### **No data showing on dashboard**

1. Check if MQTT messages are being received:

   ```bash
   docker logs forming-app | grep "Received message"
   ```

2. Publish test message:
   ```bash
   docker exec -it emqx_container_name emqx_ctl messages publish \
     production/mdcw \
     '{"ts":"2025-12-09 10:00:00","reg2":100,"reg5":1,"reg114":500,"prefix":"test"}'
   ```

### **Port already in use**

Change port mapping in `docker-compose.yml`:

```yaml
ports:
  - "3001:3000" # Change external port from 3000 to 3001
```

---

## 🔐 Security Notes

- Never commit `.env.local` to git (already in `.gitignore`)
- Change default database passwords in production
- Use MQTT authentication in production (set `MQTT_USER` and `MQTT_PASSWORD`)
- Consider using HTTPS/TLS for production deployment

---

## 📈 Performance

- **Data Limit**: Dashboard shows last 100 records per query
- **Auto-refresh**: 60 seconds
- **Summary Window**: Last 1 hour
- **Skip Log**: Automatically filters weight = 0

---

## 🚀 Development

### **Local Development (without Docker)**

```bash
# 1. Install Go 1.21+
# 2. Install dependencies
go mod download

# 3. Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export MQTT_HOST=localhost
export MQTT_PORT=1883

# 4. Run
go run .
```

### **Build Binary**

```bash
# Windows
go build -o forming.exe

# Linux
GOOS=linux GOARCH=amd64 go build -o forming
```

---

## 📝 API Endpoints

| Method | Endpoint                                | Description             |
| ------ | --------------------------------------- | ----------------------- |
| GET    | `/`                                     | Main dashboard          |
| GET    | `/data-list?status=&sort=`              | Data list with filters  |
| GET    | `/data-by-prefix?prefix=&status=&sort=` | Data filtered by prefix |
| GET    | `/summary`                              | Summary stats (1 hour)  |
| GET    | `/skip-log`                             | Skip log entries        |
| GET    | `/update-time`                          | Current server time     |

---

## ✅ Production Checklist

- [ ] `.env.local` configured with production credentials
- [ ] Database passwords changed from defaults
- [ ] MQTT authentication enabled (if needed)
- [ ] Port 3000 accessible from network
- [ ] Docker and Docker Compose installed
- [ ] `docker-compose up -d` successful
- [ ] Logs show "Connected to PostgreSQL"
- [ ] Logs show "MQTT Connected successfully"
- [ ] Dashboard accessible via browser
- [ ] Test MQTT message received and displayed

---

**Version:** 2.0  
**Last Updated:** 2025-12-09

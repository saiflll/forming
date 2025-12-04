package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	_ "github.com/lib/pq"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Payload structure matching the JSON
type Payload struct {
	Ts     string `json:"ts"`
	Weight int    `json:"weight"`
	Total  int    `json:"total"`
	Reject int    `json:"reject"`
	Status int    `json:"status"`
	Prefix string `json:"prefix"`
}

// Record structure for database rows
type Record struct {
	ID        int       `json:"id"`
	Ts        string    `json:"ts"`
	Weight    int       `json:"weight"`
	Total     int       `json:"total"`
	Reject    int       `json:"reject"`
	Status    int       `json:"status"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"created_at"`
}

var db *sql.DB

func main() {
	// --- Database Connection ---
	dbHost := getEnv("DB_HOST", "172.20.100.11")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASSWORD", "password_rahasia_anda")
	dbName := getEnv("DB_NAME", "servfi")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Error opening database connection: %v", err)
	} else {
		if err = db.Ping(); err != nil {
			log.Printf("Warning: Could not connect to database: %v", err)
		} else {
			log.Println("Connected to PostgreSQL")
			createTable()
		}
	}
	if db != nil {
		defer db.Close()
	}

	// --- MQTT Connection ---
	mqttHost := getEnv("MQTT_HOST", "172.20.100.11")
	mqttPort := getEnv("MQTT_PORT", "1883")
	brokerUrl := fmt.Sprintf("tcp://%s:%s", mqttHost, mqttPort)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerUrl)
	opts.SetClientID("forming-app-subscriber-" + fmt.Sprintf("%d", time.Now().Unix()))
	opts.SetDefaultPublishHandler(messagePubHandler)

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("MQTT Connection lost: %v", err)
	}
	opts.OnConnect = func(c mqtt.Client) {
		log.Println("MQTT Connected")
		subscribe(c)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("Warning: Could not connect to MQTT: %v", token.Error())
	}

	// --- Fiber Setup ---
	engine := html.New("./views", ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// Route Halaman Utama
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Title": "Dashboard Ringan",
			"Time":  time.Now().Format("15:04:05"),
		})
	})

	// API Endpoint untuk HTMX - Update Time
	app.Get("/update-time", func(c *fiber.Ctx) error {
		return c.SendString(time.Now().Format("15:04:05") + " WIB")
	})

	// API Endpoint untuk HTMX - Data List
	app.Get("/data-list", func(c *fiber.Ctx) error {
		status := c.Query("status")
		sortBy := c.Query("sort")

		records, err := getRecords(status, sortBy)
		if err != nil {
			log.Println("Error fetching records:", err)
			return c.Status(500).SendString("Error fetching data")
		}

		return c.Render("data_list", fiber.Map{
			"Records": records,
		})
	})

	// API Endpoint untuk Summary (Total 1 Jam Terakhir)
	app.Get("/summary", func(c *fiber.Ctx) error {
		summaries, err := getSummary()
		if err != nil {
			log.Println("Error fetching summary:", err)
			return c.Status(500).SendString("Error fetching summary")
		}
		return c.Render("summary", fiber.Map{
			"Summaries": summaries,
		})
	})

	log.Fatal(app.Listen(":3000"))
}

func messagePubHandler(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())

	var payload Payload
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		log.Println("Error parsing JSON:", err)
		return
	}

	insertData(payload)
}

func subscribe(client mqtt.Client) {
	topic := "production/mdcw"
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	if token.Error() != nil {
		log.Printf("Error subscribing to topic %s: %v", topic, token.Error())
	} else {
		log.Printf("Subscribed to topic: %s", topic)
	}
}

func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS production_mdcw (
		id SERIAL PRIMARY KEY,
		ts VARCHAR(50),
		weight INTEGER,
		total INTEGER,
		reject INTEGER,
		status INTEGER,
		prefix VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	_, err := db.Exec(query)
	if err != nil {
		log.Println("Error creating table:", err)
	} else {
		log.Println("Table 'production_mdcw' ensured")
	}

	// Migrasi manual untuk menambahkan kolom prefix jika belum ada (untuk tabel yg sudah existing)
	alterQuery := `ALTER TABLE production_mdcw ADD COLUMN IF NOT EXISTS prefix VARCHAR(50)`
	_, err = db.Exec(alterQuery)
	if err != nil {
		log.Println("Error altering table:", err)
	}
}

func insertData(p Payload) {
	if db == nil {
		return
	}

	query := `INSERT INTO production_mdcw (ts, weight, total, reject, status, prefix) VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := db.Exec(query, p.Ts, p.Weight, p.Total, p.Reject, p.Status, p.Prefix)
	if err != nil {
		log.Println("Error inserting data:", err)
	} else {
		log.Println("Data inserted successfully")
	}
}

func getRecords(statusFilter string, sortBy string) ([]Record, error) {
	if db == nil {
		return []Record{}, nil
	}

	query := "SELECT id, ts, weight, total, reject, status, prefix, created_at FROM production_mdcw WHERE 1=1"
	args := []interface{}{}
	argId := 1

	if statusFilter != "" && statusFilter != "all" {
		query += fmt.Sprintf(" AND status = $%d", argId)
		args = append(args, statusFilter)
		argId++
	}

	if sortBy == "weight_desc" {
		query += " ORDER BY weight DESC"
	} else if sortBy == "weight_asc" {
		query += " ORDER BY weight ASC"
	} else {
		query += " ORDER BY created_at DESC" // Default sort
	}

	query += " LIMIT 50" // Limit to last 50 records for display

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		// Handle NULL prefix if any
		var prefix sql.NullString

		if err := rows.Scan(&r.ID, &r.Ts, &r.Weight, &r.Total, &r.Reject, &r.Status, &prefix, &r.CreatedAt); err != nil {
			return nil, err
		}
		if prefix.Valid {
			r.Prefix = prefix.String
		} else {
			r.Prefix = "-"
		}
		records = append(records, r)
	}
	return records, nil
}

type Summary struct {
	Prefix      string
	TotalCount  int
	TotalWeight int
}

func getSummary() ([]Summary, error) {
	if db == nil {
		return []Summary{}, nil
	}

	// Hitung jumlah barang (count) dan total berat per prefix dalam 1 jam terakhir
	// Asumsi: Kita menghitung record yang masuk (status=1 OK)
	query := `
		SELECT
			COALESCE(prefix, 'Unknown') as prefix,
			COUNT(*) as total_count,
			COALESCE(SUM(weight), 0) as total_weight
		FROM production_mdcw
		WHERE created_at >= NOW() - INTERVAL '1 hour'
		GROUP BY prefix
		ORDER BY prefix ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []Summary
	for rows.Next() {
		var s Summary
		if err := rows.Scan(&s.Prefix, &s.TotalCount, &s.TotalWeight); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

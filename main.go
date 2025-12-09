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
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	_ "github.com/lib/pq"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Payload structure matching the JSON from IoT device
type Payload struct {
	Ts     string `json:"ts"`     // Timestamp: YYYY-MM-DD HH:MM:SS
	Reg2   int    `json:"reg2"`   // Total Pack Count (pieces)
	Reg5   int    `json:"reg5"`   // Status Code: 1=OK, 2=Under, 3=Over
	Reg114 int    `json:"reg114"` // Weight in grams
	Prefix string `json:"prefix"` // Node prefix identifier
}

// Record structure for database rows
type Record struct {
	ID        int       `json:"id"`
	Ts        string    `json:"ts"`
	Reg2      int       `json:"reg2"`   // Total Pack Count
	Reg5      int       `json:"reg5"`   // Status Code
	Reg114    int       `json:"reg114"` // Weight
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
	mqttUser := getEnv("MQTT_USER", "")
	mqttPass := getEnv("MQTT_PASSWORD", "")
	brokerUrl := fmt.Sprintf("tcp://%s:%s", mqttHost, mqttPort)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerUrl)
	opts.SetClientID("forming-app-subscriber-" + fmt.Sprintf("%d", time.Now().Unix()))
	opts.SetDefaultPublishHandler(messagePubHandler)

	// Connection settings
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetProtocolVersion(4) // MQTT 3.1.1

	// Set credentials if provided
	if mqttUser != "" {
		opts.SetUsername(mqttUser)
		log.Printf("MQTT Username: %s", mqttUser)
	}
	if mqttPass != "" {
		opts.SetPassword(mqttPass)
		log.Println("MQTT Password: ***")
	}

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("MQTT Connection lost: %v - Will auto-reconnect", err)
	}
	opts.OnConnect = func(c mqtt.Client) {
		log.Println("MQTT Connected successfully!")
		subscribe(c)
	}

	log.Printf("Connecting to MQTT broker: %s", brokerUrl)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("Warning: Could not connect to MQTT: %v", token.Error())
		log.Println("App will continue running. MQTT will auto-reconnect when available.")
	}

	// --- Fiber Setup ---
	engine := html.New("./views", ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Use(logger.New())
	app.Static("/public", "./public")

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

		records, err := getRecords("", status, sortBy)
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

	// API Endpoint untuk Data berdasarkan Prefix
	app.Get("/data-by-prefix", func(c *fiber.Ctx) error {
		prefix := c.Query("prefix")
		status := c.Query("status")
		sortBy := c.Query("sort")

		records, err := getRecords(prefix, status, sortBy)
		if err != nil {
			log.Println("Error fetching records:", err)
			return c.Status(500).SendString("Error fetching data")
		}

		return c.Render("data_list", fiber.Map{
			"Records": records,
		})
	})

	// API Endpoint untuk Skip Log
	app.Get("/skip-log", func(c *fiber.Ctx) error {
		skipLogs, err := getSkipLogs()
		if err != nil {
			log.Println("Error fetching skip logs:", err)
			return c.JSON([]map[string]interface{}{})
		}
		return c.JSON(skipLogs)
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
		reg2 INTEGER,
		reg5 INTEGER,
		reg114 INTEGER,
		prefix VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	_, err := db.Exec(query)
	if err != nil {
		log.Println("Error creating table:", err)
	} else {
		log.Println("Table 'production_mdcw' ensured")
	}

	// Create skip_log table for tracking skipped data (weight = 0)
	skipLogQuery := `
	CREATE TABLE IF NOT EXISTS skip_log (
		id SERIAL PRIMARY KEY,
		ts VARCHAR(50),
		reg2 INTEGER,
		reg5 INTEGER,
		reg114 INTEGER,
		prefix VARCHAR(50),
		reason VARCHAR(100),
		skipped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	_, err = db.Exec(skipLogQuery)
	if err != nil {
		log.Println("Error creating skip_log table:", err)
	} else {
		log.Println("Table 'skip_log' ensured")
	}
}

func insertData(p Payload) {
	if db == nil {
		return
	}

	// Skip data with weight = 0 and log it
	if p.Reg114 == 0 {
		log.Printf("[SKIP] Data with weight=0 from %s at %s (reg2=%d, reg5=%d)", p.Prefix, p.Ts, p.Reg2, p.Reg5)

		// Insert to skip_log
		skipQuery := `INSERT INTO skip_log (ts, reg2, reg5, reg114, prefix, reason) VALUES ($1, $2, $3, $4, $5, $6)`
		_, err := db.Exec(skipQuery, p.Ts, p.Reg2, p.Reg5, p.Reg114, p.Prefix, "Weight is zero")
		if err != nil {
			log.Println("Error logging skipped data:", err)
		}
		return
	}

	query := `INSERT INTO production_mdcw (ts, reg2, reg5, reg114, prefix) VALUES ($1, $2, $3, $4, $5)`

	_, err := db.Exec(query, p.Ts, p.Reg2, p.Reg5, p.Reg114, p.Prefix)
	if err != nil {
		log.Println("Error inserting data:", err)
	} else {
		log.Println("Data inserted successfully")
	}
}

func getRecords(prefixFilter string, statusFilter string, sortBy string) ([]Record, error) {
	if db == nil {
		return []Record{}, nil
	}

	query := "SELECT id, ts, reg2, reg5, reg114, prefix, created_at FROM production_mdcw WHERE 1=1"
	args := []interface{}{}
	argId := 1

	// Filter by prefix (optional)
	if prefixFilter != "" && prefixFilter != "all" {
		query += fmt.Sprintf(" AND prefix = $%d", argId)
		args = append(args, prefixFilter)
		argId++
	}

	// Filter by status
	if statusFilter != "" && statusFilter != "all" {
		query += fmt.Sprintf(" AND reg5 = $%d", argId)
		args = append(args, statusFilter)
		argId++
	}

	// Sorting
	if sortBy == "weight_desc" {
		query += " ORDER BY reg114 DESC"
	} else if sortBy == "weight_asc" {
		query += " ORDER BY reg114 ASC"
	} else {
		query += " ORDER BY created_at DESC" // Default sort
	}

	query += " LIMIT 100" // Limit to last 100 records for display

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		var prefix sql.NullString
		var reg2, reg5, reg114 sql.NullInt64

		if err := rows.Scan(&r.ID, &r.Ts, &reg2, &reg5, &reg114, &prefix, &r.CreatedAt); err != nil {
			return nil, err
		}

		if prefix.Valid {
			r.Prefix = prefix.String
		} else {
			r.Prefix = "-"
		}

		if reg2.Valid {
			r.Reg2 = int(reg2.Int64)
		}
		if reg5.Valid {
			r.Reg5 = int(reg5.Int64)
		}
		if reg114.Valid {
			r.Reg114 = int(reg114.Int64)
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

	// Count total records and sum of weight (reg114) per prefix in last 1 hour
	query := `
		SELECT
			COALESCE(prefix, 'Unknown') as prefix,
			COUNT(*) as total_count,
			COALESCE(SUM(reg114), 0) as total_weight
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

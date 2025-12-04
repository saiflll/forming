package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	_ "github.com/lib/pq"
)

// Payload structure matching the JSON
type Payload struct {
	Ts     string `json:"ts"`
	Weight int    `json:"weight"`
	Total  int    `json:"total"`
	Reject int    `json:"reject"`
	Status int    `json:"status"`
}

// Record structure for database rows
type Record struct {
	ID        int       `json:"id"`
	Ts        string    `json:"ts"`
	Weight    int       `json:"weight"`
	Total     int       `json:"total"`
	Reject    int       `json:"reject"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

var db *sql.DB

func main() {
	// --- Database Connection ---
	connStr := "postgres://postgres:postgres@172.20.100.11:5432/postgres?sslmode=disable"
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
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://172.20.100.11:1883")
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
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	_, err := db.Exec(query)
	if err != nil {
		log.Println("Error creating table:", err)
	} else {
		log.Println("Table 'production_mdcw' ensured")
	}
}

func insertData(p Payload) {
	if db == nil {
		return
	}

	query := `INSERT INTO production_mdcw (ts, weight, total, reject, status) VALUES ($1, $2, $3, $4, $5)`

	_, err := db.Exec(query, p.Ts, p.Weight, p.Total, p.Reject, p.Status)
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

	query := "SELECT id, ts, weight, total, reject, status, created_at FROM production_mdcw WHERE 1=1"
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
		if err := rows.Scan(&r.ID, &r.Ts, &r.Weight, &r.Total, &r.Reject, &r.Status, &r.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

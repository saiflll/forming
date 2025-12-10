package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var sheetsService *sheets.Service
var spreadsheetID string

// InitGoogleSheets initializes Google Sheets API connection
func InitGoogleSheets() error {
	// Get credentials from environment variable (base64 encoded JSON)
	credsBase64 := getEnv("GOOGLE_SHEETS_CREDENTIALS", "")
	spreadsheetID = getEnv("GOOGLE_SPREADSHEET_ID", "")

	if credsBase64 == "" || spreadsheetID == "" {
		log.Println("⚠️ Google Sheets not configured (GOOGLE_SHEETS_CREDENTIALS or GOOGLE_SPREADSHEET_ID missing)")
		return nil // Not an error, just skip
	}

	// Decode base64 credentials
	credJSON, err := base64.StdEncoding.DecodeString(credsBase64)
	if err != nil {
		return fmt.Errorf("failed to decode credentials: %v", err)
	}

	// Create Google Sheets service
	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithCredentialsJSON(credJSON))
	if err != nil {
		return fmt.Errorf("unable to create Sheets service: %v", err)
	}

	sheetsService = srv
	log.Println("✅ Google Sheets API initialized successfully")
	return nil
}

// AppendToSheet appends a row to Google Sheets
func AppendToSheet(p Payload) error {
	if sheetsService == nil {
		return nil // Sheets not configured, skip silently
	}

	// Prepare row data
	row := []interface{}{
		p.Ts,
		p.Prefix,
		p.Reg2,
		p.Reg5,
		p.Reg114,
		getStatusText(p.Reg5),
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{row},
	}

	// Append to sheet
	sheetName := getEnv("GOOGLE_SHEET_NAME", "Production Data")
	appendRange := fmt.Sprintf("%s!A:F", sheetName)

	_, err := sheetsService.Spreadsheets.Values.Append(
		spreadsheetID,
		appendRange,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("unable to append data to sheet: %v", err)
	}

	log.Printf("✅ Data exported to Google Sheets: %s - %dg", p.Prefix, p.Reg114)
	return nil
}

// getStatusText converts status code to text
func getStatusText(status int) string {
	switch status {
	case 8:
		return "Mesin Mati"
	case 90:
		return "Idle"
	case 8201:
		return "Detect Metal"
	case 25:
		return "Under Weight"
	case 73:
		return "Over Weight"
	default:
		return fmt.Sprintf("Unknown (%d)", status)
	}
}

// CreateSheetIfNotExists creates the sheet with headers if it doesn't exist
func CreateSheetIfNotExists() error {
	if sheetsService == nil {
		return nil
	}

	sheetName := getEnv("GOOGLE_SHEET_NAME", "Production Data")

	// Check if sheet already has headers
	readRange := fmt.Sprintf("%s!A1:F1", sheetName)
	resp, err := sheetsService.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil || len(resp.Values) == 0 {
		// Add headers
		headers := []interface{}{"Timestamp", "Prefix", "Pack Count", "Status Code", "Weight (g)", "Status"}
		valueRange := &sheets.ValueRange{
			Values: [][]interface{}{headers},
		}

		_, err := sheetsService.Spreadsheets.Values.Update(
			spreadsheetID,
			fmt.Sprintf("%s!A1:F1", sheetName),
			valueRange,
		).ValueInputOption("USER_ENTERED").Do()

		if err != nil {
			return fmt.Errorf("failed to create headers: %v", err)
		}
		log.Println("✅ Google Sheets headers created")
	}

	return nil
}

// Helper function to set up Google Sheets from credentials file
func SetupSheetsFromFile(credentialsPath string) error {
	credJSON, err := os.ReadFile(credentialsPath)
	if err != nil {
		return fmt.Errorf("unable to read credentials file: %v", err)
	}

	// Convert to base64 for environment variable
	credBase64 := base64.StdEncoding.EncodeToString(credJSON)
	fmt.Printf("\nAdd this to your .env.local file:\n")
	fmt.Printf("GOOGLE_SHEETS_CREDENTIALS=%s\n\n", credBase64)

	// Parse to get project info
	var cred map[string]interface{}
	if err := json.Unmarshal(credJSON, &cred); err == nil {
		if email, ok := cred["client_email"].(string); ok {
			fmt.Printf("Service Account Email: %s\n", email)
			fmt.Printf("⚠️ Make sure to share your Google Spreadsheet with this email!\n\n")
		}
	}

	return nil
}

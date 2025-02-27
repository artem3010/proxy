package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type RequestRow struct {
	InventoryID string `json:"inventoryId"`
}

type RequestBody struct {
	Rows []RequestRow `json:"rows"`
}

type EmissionsBreakdown struct {
	TotalEmissionsGrams  float64 `json:"total_emissions_grams"`
	InventoryCoverage    string  `json:"inventory_coverage"`
	ClimateRiskCompliant bool    `json:"climate_risk_compliant"`
}

type ResponseRow struct {
	InventoryID        string             `json:"inventoryId"`
	EmissionsBreakdown EmissionsBreakdown `json:"emissionsBreakdown"`
}

type ResponseBody struct {
	Rows []ResponseRow `json:"rows"`
}

func measureHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var requestBody RequestBody
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	time.Sleep(300 * time.Millisecond)

	rand.Seed(time.Now().UnixNano())
	coverageOptions := []string{"modeled", "measured"}

	var responseRows []ResponseRow
	for _, row := range requestBody.Rows {
		// Генерируем случайные данные для мока
		responseData := EmissionsBreakdown{
			TotalEmissionsGrams:  rand.Float64()*2000 + 500, // От 500 до 2500 грамм
			InventoryCoverage:    coverageOptions[rand.Intn(len(coverageOptions))],
			ClimateRiskCompliant: rand.Intn(2) == 1,
		}

		responseRows = append(responseRows, ResponseRow{
			InventoryID:        row.InventoryID,
			EmissionsBreakdown: responseData,
		})
	}

	response := ResponseBody{Rows: responseRows}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/v2/measure", measureHandler)
	log.Println("Mock server running on :8080")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

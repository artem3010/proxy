package measure_v1_dto

import "errors"

type RowsRequest struct {
	InventoryIds []InventoryId `json:"inventoryIds"`
}

type InventoryId struct {
	Id       string `json:"inventoryId"`
	Priority int    `json:"priority"`
}

func (r RowsRequest) Validate() error {
	if r.InventoryIds == nil {
		return errors.New("wrong request, missed inventory ids")
	}
	return nil
}

// EmissionsBreakdown структура для ответа
type EmissionsBreakdown struct {
	TotalEmissionsGrams  float64 `json:"total_emissions_grams"`
	InventoryCoverage    string  `json:"inventory_coverage"`
	ClimateRiskCompliant bool    `json:"climate_risk_compliant"`
}

// ResponseRow структура для выходных данных
type ResponseRow struct {
	InventoryID        string             `json:"inventoryId"`
	EmissionsBreakdown EmissionsBreakdown `json:"emissionsBreakdown"`
}

// ResponseBody структура для ответа
type ResponseBody struct {
	Rows []ResponseRow `json:"rows"`
}

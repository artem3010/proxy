package measurev1dto

import "errors"

// RowsRequest dto of measureV1 api
type RowsRequest struct {
	InventoryIds []InventoryId `json:"inventoryIds"`
}

// InventoryId inventory id with priority
type InventoryId struct {
	Id       string `json:"inventoryId"`
	Priority int    `json:"priority"`
}

// Validate validation of request
func (r RowsRequest) Validate() error {
	if r.InventoryIds == nil {
		return errors.New("wrong request, missed inventory ids")
	}
	return nil
}

// EmissionsBreakdown response structure of emission
type EmissionsBreakdown struct {
	TotalEmissionsGrams  float64 `json:"total_emissions_grams"`
	InventoryCoverage    string  `json:"inventory_coverage"`
	ClimateRiskCompliant bool    `json:"climate_risk_compliant"`
}

// ResponseRow structure fow response row
type ResponseRow struct {
	InventoryID        string             `json:"inventoryId"`
	EmissionsBreakdown EmissionsBreakdown `json:"emissionsBreakdown"`
}

// ResponseBody structure for response
type ResponseBody struct {
	Rows []ResponseRow `json:"rows"`
}

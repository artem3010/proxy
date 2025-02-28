package socopev3dto

// RequestRow dto of socopeV3 api
type RequestRow struct {
	InventoryID string `json:"inventoryId"`
	Priority    int    `json:"priority,omitempty"`
}

// RequestBody dto of socopeV3 api
type RequestBody struct {
	Rows []RequestRow `json:"rows"`
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

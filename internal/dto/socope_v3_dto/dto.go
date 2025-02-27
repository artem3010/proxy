package socope_v3_dto

type RequestRow struct {
	InventoryID string `json:"inventoryId"`
	Priority    int    `json:"priority,omitempty"`
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

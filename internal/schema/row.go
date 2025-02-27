package schema

type EmissionsBreakdown struct {
	TotalEmissionsGrams  float64
	InventoryCoverage    string
	ClimateRiskCompliant bool
}

type Row struct {
	InventoryId        string
	Priority           int
	EmissionsBreakdown EmissionsBreakdown
}

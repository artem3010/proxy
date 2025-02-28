package schema

// EmissionsBreakdown model of an emission
type EmissionsBreakdown struct {
	TotalEmissionsGrams  float64
	InventoryCoverage    string
	ClimateRiskCompliant bool
}

// Row model of an emission with priority
type Row struct {
	InventoryId        string
	Priority           int
	EmissionsBreakdown EmissionsBreakdown
}

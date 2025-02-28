package wrapper

import (
	"context"
	"fmt"
	socopev3dto "proxy/internal/dto/socope_v3_dto"
	"proxy/internal/schema"
	"time"
)

type service struct {
	emissionClient emissionClient
	timeout        time.Duration
}

// New returns a wrapper above scopeV3
func New(emissionClient emissionClient,
	timeout time.Duration,
) *service {
	return &service{
		emissionClient: emissionClient,
		timeout:        timeout,
	}
}

func (s *service) GetEmissions(ctx context.Context, inventoryIds []schema.Row) ([]schema.Row, error) {
	if len(inventoryIds) == 0 {
		return []schema.Row{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	emissions, err := s.emissionClient.FetchEmissions(ctx, toDto(inventoryIds))
	if err != nil {
		return nil, fmt.Errorf("error in emission client request %s", err)
	}

	result := toSchema(emissions)

	return result, nil
}

func toSchema(emissions *socopev3dto.ResponseBody) []schema.Row {
	rows := make([]schema.Row, 0, len(emissions.Rows))

	for _, val := range emissions.Rows {
		rows = append(rows, schema.Row{
			InventoryId: val.InventoryID,
			EmissionsBreakdown: schema.EmissionsBreakdown{
				TotalEmissionsGrams:  val.EmissionsBreakdown.TotalEmissionsGrams,
				InventoryCoverage:    val.EmissionsBreakdown.InventoryCoverage,
				ClimateRiskCompliant: val.EmissionsBreakdown.ClimateRiskCompliant,
			},
		})
	}
	return rows
}

func toDto(ids []schema.Row) socopev3dto.RequestBody {

	rows := make([]socopev3dto.RequestRow, 0, len(ids))

	for _, val := range ids {
		rows = append(rows, socopev3dto.RequestRow{
			InventoryID: val.InventoryId,
			Priority:    val.Priority,
		})
	}

	return socopev3dto.RequestBody{
		Rows: rows,
	}
}

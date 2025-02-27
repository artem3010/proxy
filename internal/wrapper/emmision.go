package wrapper

import (
	"context"
	"proxy/internal/dto/socope_v3_dto"
	"proxy/internal/schema"
	"time"
)

type Service struct {
	emissionClient emissionClient
	timeout        time.Duration
}

func New(emissionClient emissionClient,
	timeout time.Duration,
) *Service {
	return &Service{
		emissionClient: emissionClient,
		timeout:        timeout,
	}
}

func (s *Service) GetEmissions(ctx context.Context, inventoryIds []schema.Row) ([]schema.Row, error) {
	if len(inventoryIds) == 0 {
		return []schema.Row{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	emissions, err := s.emissionClient.FetchEmissions(ctx, toDto(inventoryIds))
	if err != nil {
		return nil, err
	}

	result := toSchema(emissions)

	return result, nil
}

func toSchema(emissions *socope_v3_dto.ResponseBody) []schema.Row {
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

func toDto(ids []schema.Row) socope_v3_dto.RequestBody {

	rows := make([]socope_v3_dto.RequestRow, 0, len(ids))

	for _, val := range ids {
		rows = append(rows, socope_v3_dto.RequestRow{
			InventoryID: val.InventoryId,
			Priority:    val.Priority,
		})
	}

	return socope_v3_dto.RequestBody{
		Rows: rows,
	}
}

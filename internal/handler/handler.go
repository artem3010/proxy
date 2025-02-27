package handler

import (
	"context"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"net/http"
	"proxy/internal/dto/measure_v1_dto"
	"proxy/internal/schema"
	"time"
)

type Handler struct {
	measureGetter  measureGetter
	requestTimeout time.Duration
}

func New(measureGetter measureGetter, timeout int64) *Handler {
	return &Handler{
		measureGetter:  measureGetter,
		requestTimeout: time.Duration(timeout) * time.Millisecond,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var request measure_v1_dto.RowsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := request.Validate(); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.requestTimeout)
	defer cancel()

	reqStart := time.Now()
	result, err := h.measureGetter.Get(ctx, toModel(request.InventoryIds))
	latency := time.Since(reqStart)

	log.Info().Str("latency", latency.String()).Msg("get latency")

	if err != nil {
		log.Err(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := &measure_v1_dto.ResponseBody{Rows: convert(result)}

	json.NewEncoder(w).Encode(response)
}

func toModel(ids []measure_v1_dto.InventoryId) map[string]schema.Row {
	result := make(map[string]schema.Row, len(ids))

	for _, val := range ids {
		result[val.Id] = schema.Row{
			InventoryId: val.Id,
			Priority:    val.Priority,
		}
	}
	return result
}

func convert(rows []schema.Row) []measure_v1_dto.ResponseRow {
	result := make([]measure_v1_dto.ResponseRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, measure_v1_dto.ResponseRow{
			InventoryID: row.InventoryId,
			EmissionsBreakdown: measure_v1_dto.EmissionsBreakdown{
				TotalEmissionsGrams:  row.EmissionsBreakdown.TotalEmissionsGrams,
				InventoryCoverage:    row.EmissionsBreakdown.InventoryCoverage,
				ClimateRiskCompliant: row.EmissionsBreakdown.ClimateRiskCompliant,
			},
		})
	}
	return result
}

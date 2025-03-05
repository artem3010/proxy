package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"proxy/internal/dto/measure_v1_dto"
	"proxy/internal/schema"
	"time"

	"github.com/rs/zerolog/log"
)

type handler struct {
	measureGetter  measureGetter
	requestTimeout time.Duration
}

// New return new handler
func New(measureGetter measureGetter, timeout time.Duration) *handler {
	return &handler{
		measureGetter:  measureGetter,
		requestTimeout: timeout,
	}
}

// Handle returns emissions by id, can set priorities in request
func (h *handler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var request measurev1dto.RowsRequest
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
		log.Err(fmt.Errorf("couldn't get emissions, %s", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := &measurev1dto.ResponseBody{Rows: convert(result)}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Err(fmt.Errorf("couldn't encode response, %s", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func toModel(ids []measurev1dto.InventoryId) map[string]schema.Row {
	result := make(map[string]schema.Row, len(ids))

	for _, val := range ids {
		result[val.Id] = schema.Row{
			InventoryId: val.Id,
			Priority:    val.Priority,
		}
	}
	return result
}

func convert(rows []schema.Row) []measurev1dto.ResponseRow {
	result := make([]measurev1dto.ResponseRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, measurev1dto.ResponseRow{
			InventoryID: row.InventoryId,
			EmissionsBreakdown: measurev1dto.EmissionsBreakdown{
				TotalEmissionsGrams:  row.EmissionsBreakdown.TotalEmissionsGrams,
				InventoryCoverage:    row.EmissionsBreakdown.InventoryCoverage,
				ClimateRiskCompliant: row.EmissionsBreakdown.ClimateRiskCompliant,
			},
		})
	}
	return result
}

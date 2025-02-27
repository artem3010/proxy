package wrapper

import (
	"context"
	"proxy/internal/dto/socope_v3_dto"
)

type emissionClient interface {
	FetchEmissions(ctx context.Context, request socope_v3_dto.RequestBody) (*socope_v3_dto.ResponseBody, error)
}

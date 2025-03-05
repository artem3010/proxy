package wrapper

import (
	"context"
	socopev3dto "proxy/internal/dto/socope_v3_dto"
)

type emissionClient interface {
	FetchEmissions(ctx context.Context, request socopev3dto.RequestBody) (*socopev3dto.ResponseBody, error)
}

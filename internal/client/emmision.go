package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	socopev3dto "proxy/internal/dto/socope_v3_dto"
	"time"
)

type client struct {
	APIURL string
	client *http.Client
}

// NewClient returns new client
func NewClient(apiURL string, timeout time.Duration) (*client, error) {
	if apiURL == "" {
		return nil, errors.New("api url is empty")
	}
	return &client{
		APIURL: apiURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// FetchEmissions get emissions
func (c *client) FetchEmissions(ctx context.Context, request socopev3dto.RequestBody) (*socopev3dto.ResponseBody, error) {
	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("err during marshaling of a request: %s", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.APIURL, bytes.NewBuffer(requestData))
	if err != nil {
		return nil, fmt.Errorf("err during creating a request with context: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %s", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error().Msg("couldn't close a body")
			return
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected status code: " + resp.Status)
	}

	var response socopev3dto.ResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("err during unmarshaling of a response: %s", err)
	}

	return &response, nil
}

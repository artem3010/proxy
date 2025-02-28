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
	"proxy/internal/dto/socope_v3_dto"
	"time"
)

type Client struct {
	APIURL string
	client *http.Client
}

func NewClient(apiURL string, timeout time.Duration) (*Client, error) {
	if apiURL == "" {
		return nil, errors.New("api url is empty")
	}
	return &Client{
		APIURL: apiURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *Client) FetchEmissions(ctx context.Context, request socope_v3_dto.RequestBody) (*socope_v3_dto.ResponseBody, error) {
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

	var response socope_v3_dto.ResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("err during unmarshaling of a response: %s", err)
	}

	return &response, nil
}

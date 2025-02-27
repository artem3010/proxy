package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"proxy/internal/dto/socope_v3_dto"
)

const defaultAPIURL = "http://localhost:8080/v2/measure"

type Client struct {
	APIURL string
	client *http.Client
}

func NewClient(apiURL string) *Client {
	if apiURL == "" {
		apiURL = defaultAPIURL
	}
	return &Client{
		APIURL: apiURL,
		client: &http.Client{},
	}
}

func (c *Client) FetchEmissions(ctx context.Context, request socope_v3_dto.RequestBody) (*socope_v3_dto.ResponseBody, error) {
	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.APIURL, bytes.NewBuffer(requestData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected status code: " + resp.Status)
	}

	var response socope_v3_dto.ResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

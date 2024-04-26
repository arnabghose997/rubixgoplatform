package client

import (
	"github.com/rubixchain/rubixgoplatform/core/model"
	"github.com/rubixchain/rubixgoplatform/setup"
)

func (c *Client) FetchPartTokens(req model.FetchPartTokensRequest) (*model.FetchPartTokensResponse, error) {
	var resp model.FetchPartTokensResponse
	err := c.sendJSONRequest("GET", setup.APIFetchPartTokens, nil, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

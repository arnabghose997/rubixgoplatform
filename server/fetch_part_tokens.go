package server

import (
	"net/http"

	"github.com/rubixchain/rubixgoplatform/core/model"
	"github.com/rubixchain/rubixgoplatform/wrapper/ensweb"
)

func (s *Server) APIFetchPartTokens(req *ensweb.Request) *ensweb.Result {
	var fetchPartTokensRequest model.FetchPartTokensRequest
	err := s.ParseJSON(req, &fetchPartTokensRequest)
	if err != nil {
		return s.BasicResponse(req, false, "Invalid input", nil)
	}
	response := s.c.FetchPartTokens(&fetchPartTokensRequest)
	if !response.Status {
		return s.RenderJSON(req, response.BasicResponse, http.StatusOK)
	}

	result := struct {
		Tokens []string `json:"tokens"`
		Amount float64 `json:"amount"`
	} {
		Tokens: response.Tokens,
		Amount: response.Amount,
	}

	return s.RenderJSON(req, result, http.StatusOK)
}

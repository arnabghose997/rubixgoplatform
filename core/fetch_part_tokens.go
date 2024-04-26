package core

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rubixchain/rubixgoplatform/core/model"
	"github.com/rubixchain/rubixgoplatform/core/wallet"
	"github.com/rubixchain/rubixgoplatform/wrapper/ensweb"
)

func (c *Core) PartTokenService() {
	c.l.AddRoute(APIGetPartTokensFromPeers, "GET", c.getPartTokensFromPeers)
}

func getTokenHashesFromTokens(tokens []wallet.Token) []string {
	var tokenHashes []string = make([]string, 0)
	for _, token := range tokens {
		tokenHashes = append(tokenHashes, token.TokenID)
	}
	return tokenHashes
}

func getPeerIdAndDIDFromAddress(addr string) (string, string) {
	elems := strings.Split(addr, ".")
	peerId := elems[0]
	did := elems[1]
	return peerId, did
}

func calculatePartTokenSum(tokens []wallet.Token) float64 {
	var result float64 = 0.0

	for _, token := range tokens {
		result += token.TokenValue
	}

	return result
}

func (c *Core) FetchPartTokens(req *model.FetchPartTokensRequest) *model.FetchPartTokensResponse {
	response := &model.FetchPartTokensResponse{
		BasicResponse: model.BasicResponse{
			Status: false,
		},
	}

	inputAddr := req.Address
	inputPeerId, inputDid := getPeerIdAndDIDFromAddress(inputAddr)

	var partTokens []wallet.Token
	// Check if the provided peerID is same as the client's PeerID
	if inputPeerId == c.peerID {
		partTokens, err := c.w.ReadAllPartTokens(inputDid)
		if err != nil {
			if strings.Contains(err.Error(), "no records found") {
				response.Status = true
				response.Message = ""
				response.Result = &model.FetchPartTokensResponse{
					Tokens: make([]string, 0),
					Amount: 0.0,
				}
				response.Amount = 0.0
				response.Tokens = make([]string, 0)
				return response
			} else {
				errMsg := fmt.Sprintf("error occurred while fetching part tokens, err: %v", err.Error())
				response.Message = errMsg
				c.log.Error(errMsg)
				return response
			}
		}

		if len(partTokens) == 0 {
			response.Status = true
			response.Message = ""
			response.Result = &model.FetchPartTokensResponse{
				Tokens: make([]string, 0),
				Amount: 0.0,
			}
			response.Amount = 0.0
			response.Tokens = make([]string, 0)
			return response
		} else {
			response.Status = true
			response.Message = ""
			response.Result = &model.FetchPartTokensResponse{
				Tokens: getTokenHashesFromTokens(partTokens),
				Amount: calculatePartTokenSum(partTokens),
			}
			response.Amount = calculatePartTokenSum(partTokens)
			response.Tokens = getTokenHashesFromTokens(partTokens)
			return response
		}

	} else {
		peer, err := c.getPeer(inputAddr)
		if err != nil {
			errMsg := fmt.Sprintf("unable to connect to peer %v, err: %v", inputPeerId, err.Error())
			response.Message = errMsg
			c.log.Error(errMsg)
			return response
		}

		var getPartTokensFromPeersRequest *model.GetPartTokensFromPeersRequest = &model.GetPartTokensFromPeersRequest{
			Did: inputDid,
		}
		var getPartTokensFromPeersResponse *model.GetPartTokensFromPeersResponse
		errJsonRequest := peer.SendJSONRequest("GET", APIGetPartTokensFromPeers, nil, getPartTokensFromPeersRequest, &getPartTokensFromPeersResponse, true)
		if errJsonRequest != nil {
			errMsg := fmt.Sprintf("unable to send request, err: %v", errJsonRequest)
			c.log.Error(errMsg)
			response.Message = errMsg
			return response
		}
		if !getPartTokensFromPeersResponse.Status {
			errMsg := fmt.Sprintf("unable to fetch part tokens from Peer, err: %v", response.Message)
			c.log.Error(errMsg)
			return response
		}

		tokensFromPeer := getPartTokensFromPeersResponse.Tokens
		partTokens = append(partTokens, tokensFromPeer...)
		partTokensSum := calculatePartTokenSum(partTokens)

		response.Status = true
		response.Message = ""
		response.Result = &model.FetchPartTokensResponse{
			Tokens: getTokenHashesFromTokens(partTokens),
			Amount: partTokensSum,
		}
		response.Amount = partTokensSum
		response.Tokens = getTokenHashesFromTokens(partTokens)
		return response

	}
}

func (c *Core) getPartTokensFromPeers(req *ensweb.Request) *ensweb.Result {
	response := &model.GetPartTokensFromPeersResponse{
		BasicResponse: model.BasicResponse{
			Result: false,
		},
	}

	var getPartTokensFromPeersRequest *model.GetPartTokensFromPeersRequest
	err := c.l.ParseJSON(req, &getPartTokensFromPeersRequest)
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse json request, err: %v", err.Error())
		c.log.Error(errMsg)
		response.Message = errMsg
		return c.l.RenderJSON(req, &response, http.StatusOK)
	}
	did := getPartTokensFromPeersRequest.Did

	partTokens, err := c.w.ReadAllPartTokens(did)
	if err != nil {
		if strings.Contains(err.Error(), "no records found") {
			response.Status = true
			response.Tokens = make([]wallet.Token, 0)
			return c.l.RenderJSON(req, &response, http.StatusOK)
		} else {
			errMsg := fmt.Sprintf("error occurred while fetching part tokens, err: %v", err.Error())
			response.Message = errMsg
			c.log.Error(errMsg)
			return c.l.RenderJSON(req, &response, http.StatusOK)
		}
	}

	if len(partTokens) == 0 {
		response.Status = true
		response.Tokens = make([]wallet.Token, 0)
		return c.l.RenderJSON(req, &response, http.StatusOK)
	} else {
		response.Status = true
		response.Tokens = partTokens
		return c.l.RenderJSON(req, &response, http.StatusOK)
	}
	
}

package server

import (
	"regexp"
	"strings"

	"github.com/rubixchain/rubixgoplatform/core/model"
	"github.com/rubixchain/rubixgoplatform/wrapper/ensweb"
)

func (s *Server) APISetupQuorum(req *ensweb.Request) *ensweb.Result {
	var qs model.QuorumSetup
	err := s.ParseJSON(req, &qs)
	if err != nil {
		return s.BasicResponse(req, false, "Failed to parse the input", nil)
	}
	is_alphanumeric := regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString(qs.DID)
	if !strings.HasPrefix(qs.DID, "bafybmi") || len(qs.DID) != 59 || !is_alphanumeric {
		s.log.Error("Invalid DID")
		return s.BasicResponse(req, false, "Invalid DID", nil)
	}
	err = s.c.SetupQuorum(qs.DID, qs.Password, qs.PrivKeyPassword)
	if err != nil {
		return s.BasicResponse(req, false, "Failed to setup quorum, "+err.Error(), nil)
	}
	return s.BasicResponse(req, true, "Quorum setup done successfully", nil)
}

func (s *Server) APIUpdateTransTokensInQuorumsTable(req *ensweb.Request) *ensweb.Result {
	var quorumDID string
	err := s.ParseJSON(req, &quorumDID)
	if err != nil {
		return s.BasicResponse(req, false, "Failed to parse the quorum did", nil)
	}
	is_alphanumeric := regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString(quorumDID)
	if !strings.HasPrefix(quorumDID, "bafybmi") || len(quorumDID) != 59 || !is_alphanumeric {
		s.log.Error("Invalid DID")
		return s.BasicResponse(req, false, "Invalid DID", nil)
	}
	resp := s.c.APIUpdateTransTokensInQuorumsTable(quorumDID)
	return s.BasicResponse(req, resp.Status, resp.Message, nil)
}

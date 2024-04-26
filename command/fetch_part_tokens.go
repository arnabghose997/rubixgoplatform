package command

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rubixchain/rubixgoplatform/core/model"
)

func (cmd *Command) fetchPartTokensCmd() {
	if len(strings.Split(cmd.address, ".")) != 2 {
		cmd.log.Error("Input address must be in <peerId>.<did> format")
		return
	}
	
	request := model.FetchPartTokensRequest{
		Address:   cmd.address,
	}

	response, err := cmd.c.FetchPartTokens(request)
	if err != nil {
		cmd.log.Error("unable to fetch part tokens, err: ", err)
		return
	}

	result := struct {
		Tokens []string `json:"tokens"`
		Amount float64 `json:"amount"`
	} {
		Tokens: response.Tokens,
		Amount: response.Amount,
	}

	resultBytes, err := json.MarshalIndent(result, "", " ")
	if err != nil {
		cmd.log.Error("failed to marshal result, error: %v", err)
	}

	_, err = fmt.Fprint(os.Stdout, string(resultBytes), "\n")
	if err != nil {
		cmd.log.Error(err.Error())
	}
}
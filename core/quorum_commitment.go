package core

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	"github.com/rubixchain/rubixgoplatform/core/model"
	"github.com/rubixchain/rubixgoplatform/core/wallet"
	"github.com/rubixchain/rubixgoplatform/util"
	"github.com/rubixchain/rubixgoplatform/wrapper/ensweb"
)

type QuorumCommitRequest struct {
	UserDID string `json:"user_did"`
}

type QuorumCommitResponse struct {
	model.BasicResponse
	UserDID         string `json:"user_did"`
	QuorumDID       string `json:"quorum_did"`
	CommitmentHash  string `json:"commitment_hash"`
	QuorumSignature string `json:"quorum_signature"`
}

// quorum commitment API response
func (c *Core) quorumCommitmentResponse(req *ensweb.Request) *ensweb.Result {
	quorumDID := c.l.GetQuerry(req, "did")
	commitResponse := &QuorumCommitResponse{
		QuorumDID: quorumDID,
		BasicResponse: model.BasicResponse{
			Status: false,
		},
	}
	var commitRequest QuorumCommitRequest
	err := c.l.ParseJSON(req, &commitRequest)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to parse commit request; err: %v", err)
		c.log.Error(errMsg)
		commitResponse.Message = errMsg
		return c.l.RenderJSON(req, &commitResponse, http.StatusOK)
	}
	commitResponse = c.QuorumCommitment(commitRequest.UserDID, quorumDID)
	return c.l.RenderJSON(req, &commitResponse, http.StatusOK)
}

// quorum signature on commitment hash
func (c *Core) QuorumCommitment(userDID, quorumDID string) *QuorumCommitResponse {
	commitResponse := &QuorumCommitResponse{
		QuorumDID: quorumDID,
		BasicResponse: model.BasicResponse{
			Status: false,
		},
	}
	quorumSignInterface, ok := c.qc[quorumDID]
	if !ok {
		c.log.Error("Failed to setup quorum crypto to sign on commitment hash")
		commitResponse.Message = "Failed to setup quorum crypto to sign on commitment hash"
		return commitResponse
	}

	commitmentHash, err := c.QuorumCommitmentHash(userDID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed create commitment hash; err : %v", err)
		c.log.Error(errMsg)
		commitResponse.Message = errMsg
		return commitResponse
	}

	quorumSig, err := quorumSignInterface.PvtSign(commitmentHash)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to do signature; err: %v", err)
		c.log.Error(errMsg)
		commitResponse.Message = errMsg
		return commitResponse
	}

	// store commitment hash in table
	currentTime := time.Now()
	commitmentMap := wallet.CommitmentHashMap{
		UserDID:        userDID,
		CommitmentHash: util.HexToStr(commitmentHash),
		Epoch:          int(currentTime.Unix()),
	}

	err = c.w.AddCommitmentMap(commitmentMap)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to store commitment hash for user %s; err: %v", userDID, err)
		c.log.Error(errMsg)
		commitResponse.Message = errMsg
		return commitResponse
	}

	commitResponse.UserDID = userDID
	commitResponse.CommitmentHash = util.HexToStr(commitmentHash)
	commitResponse.QuorumSignature = util.HexToStr(quorumSig)
	commitResponse.Status = true
	commitResponse.Message = "successfully generated commitment hash and signature"
	return commitResponse
}

// this function is for quorums to commit the transaction-ids of an user for which they are pledging currently
func (c *Core) QuorumCommitmentHash(userDID string) ([]byte, error) {
	// fetch transaction-ids and epoch of all these trans-tokens
	var txList []wallet.TxnEpoch

	// collect trans tokens currently pledging for : 2 ways :
	// 1. get trans-tokens from TokensTable with status 20
	transTokens, err := c.w.GetTransTokensBeingPledgedByDID(userDID)
	if err != nil {
		return nil, err
	}
	if transTokens == nil {
		// 2. If there are no tokens with status 20, read latest blocks of all tokens from level db and
		//    search the transaction-id in TokenStateHashTable
		txList, err = c.w.GetPledgingTransactionsFromLevelDB(c.testNet, userDID)
		if err != nil {
			c.log.Error("err ", err)
			return nil, err
		}
	} else {
		txList, err = c.getTxIdsQuorumIsPledgingFor(transTokens, userDID)
		if err != nil {
			errMsg := fmt.Sprintf("failed to fetch transaction ids, for which quorum is pledging currently, err : %v", err)
			return nil, fmt.Errorf("%v", errMsg)
		}
	}

	// order all the transaction ids as per epoch in ascending order
	orderedTxnList := c.w.OrderTxnIdsWithEpoch(txList)

	// hash the transactions recursively
	commitmentHash := c.recursiveHashChain(orderedTxnList)

	return commitmentHash, nil
}

// hashes txn-ids recursively in the provided order and returns the final output
func (c *Core) recursiveHashChain(txs []wallet.TxnEpoch) []byte {
	var prev []byte

	for _, tx := range txs {
		h := sha256.New()                 //--- check if already exists
		h.Write(prev)                     // previous hash
		h.Write([]byte(tx.TransactionId)) // current tx id
		prev = h.Sum(nil)
	}

	return prev
}

// manage trans-tokens, store trans-tokens with status 20 in Tokens table only if they are being pledged by the quorum currently
func (c *Core) getTxIdsQuorumIsPledgingFor(transTokensList []wallet.Token, userDID string) ([]wallet.TxnEpoch, error) {
	txnList := make([]wallet.TxnEpoch, 0)
	removeTransTokensList := make([]wallet.Token, 0)

	for _, transToken := range transTokensList {
		// 1. check latest block of each trans-token
		tokenType := RBTString
		if transToken.TokenValue < 1.0 {
			tokenType = PartString
		}
		latestBlock := c.w.GetLatestTokenBlock(transToken.TokenID, c.TokenType(tokenType))
		if latestBlock == nil {
			removeTransTokensList = append(removeTransTokensList, transToken)
			continue
		}
		//	2. get txn-id
		txnId := latestBlock.GetTid()
		// if transaction-id is empty in latest block, then remove the trans-token from TokensTable
		if txnId == "" {
			removeTransTokensList = append(removeTransTokensList, transToken)
			continue
		}
		// 3. check if txn-id is there in the TokenStateHash table
		tokenStateHashListByTxId, err := c.w.GetTokenStateHashByTransactionID(txnId)
		if err != nil {
			errMsg := fmt.Sprintf("failed to read TokenStateHash table, err : %v", err)
			return nil, fmt.Errorf("%v", errMsg)
		}
		// 4. If it is not there remove the token from TokensTable
		if tokenStateHashListByTxId == nil {
			removeTransTokensList = append(removeTransTokensList, transToken)
			continue
		}

		// check if txn id stored in table, matches with the one in latest block
		// update the txn_id and owner did if it doesn't match
		if transToken.TransactionID != txnId {
			transToken.TransactionID = txnId
			transToken.DID = latestBlock.GetOwner()
			_ = c.w.UpdateToken(&transToken)
		}
		// add it to txn list to commit
		txnList = append(txnList, wallet.TxnEpoch{
			TransactionId: txnId,
			Epoch:         latestBlock.GetEpoch(),
		})

	}

	// remove all trans-tokens from TokensTable, which are not being pledged anymore
	err := c.w.RemoveTokens(removeTransTokensList)
	if err != nil {
		// DO NOT RETURN ERROR, return txn list
		errMsg := fmt.Sprintf("failed to remove trans-tokens from TokensTbale that are unpledged by quorum , err : %v", err)
		c.log.Error(errMsg)
	}

	return txnList, nil
}

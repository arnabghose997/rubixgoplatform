package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/rubixchain/rubixgoplatform/core/storage"
	"github.com/rubixchain/rubixgoplatform/core/wallet"
	"github.com/rubixchain/rubixgoplatform/util"
	"github.com/rubixchain/rubixgoplatform/wrapper/logger"
)

const (
	QuorumTypeOne int = iota + 1
	QuorumTypeTwo
)

const (
	QuorumStorage string = "quorummanager"
)

const (
	GenericIssue int = iota
	ParentTokenNotBurned
	TokenChainNotSynced
)

type QuorumDIDPeerMap struct {
	DID         string `gorm:"column:did;primaryKey"`
	DIDType     *int   `gorm:"column:did_type"`
	PeerID      string `gorm:"column:peer_id"`
	DIDLastChar string `gorm:"column:did_last_char"`
}

type QuorumManager struct {
	ql  []string
	s   storage.Storage
	log logger.Logger
}

type QuorumData struct {
	Type    int    `gorm:"column:type" json:"type"`
	Address string `gorm:"column:address;primaryKey" json:"address"`
}

// isOldAddressFormat checks if the address is in <peerID>.<did> format (followed in versions v0.0.17 and before)
func isOldAddressFormat(address string) bool {
	return len(strings.Split(address, ".")) == 2
}

func NewQuorumManager(s storage.Storage, log logger.Logger) (*QuorumManager, error) {
	qm := &QuorumManager{
		s:   s,
		log: log.Named("quorum_manager"),
	}
	err := qm.s.Init(QuorumStorage, &QuorumData{}, true)
	if err != nil {
		qm.log.Error("Failed to init quorum storage", "err", err)
		return nil, err
	}
	var qd []QuorumData
	err = qm.s.Read(QuorumStorage, &qd, "type=?", QuorumTypeTwo)
	if err == nil {
		qm.ql = make([]string, 0)
		for _, q := range qd {
			// Node with version v0.0.17 or prior will have stored the addresses in
			// <peer ID>.<did> format. To make it compatible the current implementation,
			// we check if its in the prior format, and if its so, then we change it to
			// <did> format and update it in quorummanager table
			if isOldAddressFormat(q.Address) {
				quorumAddressElements := strings.Split(q.Address, ".")
				quorumDID := quorumAddressElements[1]

				// Replace the old address format with new format in quorummanager
				var updatedQuorumDetails QuorumData = QuorumData{
					Type:    q.Type,
					Address: quorumDID,
				}
				err = qm.s.Write(QuorumStorage, &updatedQuorumDetails)
				if err != nil {
					return nil, fmt.Errorf("failed while writing quorum info with new address format in quorummanager table, err: %v", err)
				}

				err := qm.s.Delete(QuorumStorage, &QuorumData{}, "address=?", q.Address)
				if err != nil {
					return nil, fmt.Errorf("failed while deleting quorum info to replace with new address format in quorummanager table, err: %v", err)
				}

				qm.ql = append(qm.ql, quorumDID)
			} else {
				qm.ql = append(qm.ql, q.Address)
			}
		}
	}
	return qm, nil
}

// GetQuorum will get the configured or available quorum
func (qm *QuorumManager) GetQuorum(t int, lastChar string, selfPeer string) []string {
	//QuorumTypeOne is to select quorums from the public pool of quorums instead of a private subnet.
	//Once a new node is created, it will create a DID. Using the command "registerdid", the peerID and DID will be
	//published in the network, and all the nodes listening to the subscription will have the DID added on the DIDPeerTable
	//A new variable quorumList is created, which will contain all the nodes which has DID with same last character as Transaction ID.
	//It would throw an error if it cannot find any relevant data of if the number of nodes is less than 5.
	//Then a separate array of type String called quorumAddrList, which will simply contain the address of nodes i.e. PeerID.DID
	//"quorumAddrList" is returned and checking the availability of nodes would be done in initiateConsensus function in quorum_initiator.go.
	switch t {
	case QuorumTypeOne:
		var quorumList []wallet.DIDPeerMap
		err := qm.s.Read(wallet.DIDPeerStorage, &quorumList, "did_last_char=?", lastChar)
		if err != nil {
			qm.log.Error("Quorums not present")
			return nil
		}
		if len(quorumList) < 5 {
			qm.log.Error("Not enough quorums present")
			return nil
		}
		var quorumAddrList []string
		quorumAddrCount := 0
		for _, q := range quorumList {
			addr := string(q.PeerID + "." + q.DID)
			quorumAddrList = append(quorumAddrList, addr)
			quorumAddrCount = quorumAddrCount + 1
			if quorumAddrCount == 7 {
				break
			}
		}
		return quorumAddrList
	case QuorumTypeTwo:
		var quorumAddrList []string
		quorumAddrCount := 0
		for _, q := range qm.ql {
			peerID := qm.GetPeerID(q, selfPeer)
			addr := string(peerID + "." + q)
			quorumAddrList = append(quorumAddrList, addr)
			quorumAddrCount = quorumAddrCount + 1
			if quorumAddrCount == 7 {
				break
			}
		}
		return quorumAddrList
	}
	return nil
}

func (qm *QuorumManager) AddQuorum(qds []QuorumData) error {
	str := make([]string, 0)
	for _, qd := range qds {
		err := qm.s.Write(QuorumStorage, &qd)
		if err != nil {
			qm.log.Error("Failed to write to quorum storage", "err", err)
			return err
		}
		str = append(str, qd.Address)
	}
	qm.ql = str
	return nil
}

func (qm *QuorumManager) RemoveAllQuorum(t int) error {
	err := qm.s.Delete(QuorumStorage, &QuorumData{}, "type=?", t)
	if err != nil {
		qm.log.Error("Failed to delete quorum data", "err", err)
	}
	return err
}

func (qm *QuorumManager) GetPeerID(did string, selfPeer string) string {
	var dm QuorumDIDPeerMap
	err := qm.s.Read(wallet.DIDPeerStorage, &dm, "did=?", did)
	if err != nil && strings.Contains(err.Error(), "no records found") {
		// Check if the Quorum DID is part of the same node by looking in DIDTable
		var dt wallet.DIDType
		err2 := qm.s.Read(wallet.DIDStorage, &dt, "did=?", did)
		if err2 != nil {
			return ""
		} else {
			return selfPeer
		}
	} else {
		return dm.PeerID
	}
}

func (c *Core) AddFaucetQuorums() {
	resp, err := http.Get("http://103.209.145.177:3999/api/get-faucet-quorums")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var faucetQuorumList []string
	// {"p1.d1", "p2.d2", "p3.d3", "p4.d4", "p5.d5"}

	body, err := io.ReadAll(resp.Body)
	// Populating the tokendetail with current token number and current token level received from Faucet.
	json.Unmarshal(body, &faucetQuorumList)
	if err != nil {
		return
	}

	if len(faucetQuorumList) < 5 {
		c.log.Error("Length of Quorum List is less than Min Quorum Count(5)")
		return
	}
	var qds []QuorumData
	for _, quorum := range faucetQuorumList {
		peerID, did, _ := util.ParseAddress(quorum)
		c.w.AddDIDPeerMap(did, peerID, 4)
		qd := QuorumData{
			Type:    2,
			Address: did,
		}
		qds = append(qds, qd)
	}
	c.RemoveAllQuorum()
	c.qm.AddQuorum(qds)
	// Save to local JSON file
	err = saveQuorumsToFile(qds, "faucet_quorumlist.json")
	if err != nil {
		return
	}
}
func saveQuorumsToFile(qds []QuorumData, fileName string) error {
	// Get the current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ") // Pretty print JSON
	if err := encoder.Encode(qds); err != nil {
		return fmt.Errorf("failed to write JSON to file: %w", err)
	}
	fmt.Printf("Quorum file saved successfully at %s\n", currentDir)
	return nil
}

// this function is for quorums to commit the transaction-ids of an user for which they are pledging currently
func (c *Core) QuorumCommitment(userDID string) (string, error) {
	// fetch transaction-ids and epoch of all these trans-tokens
	txList := make([]TxnEpoch, 0)

	// collect trans tokens currently pledging for : 2 ways :
	// 1. get trans-tokens from TokensTable with status 20
	transTokens, err := c.w.GetTransTokensBeingPledged(userDID)
	if err != nil {
		return "", err
	}
	if transTokens == nil {
		// 2. If there are no tokens with status 20, read latest blocks of all tokens from level db and
		//    search the transaction-id in TokenStateHashTable

	} else {
		txList, err = c.getTxIdsQuorumIsPledgingFor(transTokens, userDID)
		if err != nil {
			errMsg := fmt.Sprintf("failed to fetch transaction ids, for which quorum is pledging currently, err : %v", err)
			return "", fmt.Errorf("%v", errMsg)
		}
	}

	// order all the transaction ids as per epoch in ascending order
	sort.Slice(txList, func(i, j int) bool {
		return txList[i].TxnEpoch < txList[j].TxnEpoch
	})

	// hash the transactions recursively
	commitmentHash := c.recursiveHashChain(txList)

	return commitmentHash, nil
}

type TxnEpoch struct {
	TransactionId string `json:"transaction_id"`
	TxnEpoch      int    `json:"epoch"`
}

// order txn ids with epoch
func (c *Core) OrderTxnIdsWithEpoch(txList []TxnEpoch) ([]TxnEpoch, error) {
	sort.Slice(txList, func(i, j int) bool {
		return txList[i].TxnEpoch < txList[j].TxnEpoch
	})
	return txList, nil
}

// hashes txn-ids recursively in the provided order and returns the final output
func (c *Core) recursiveHashChain(txs []TxnEpoch) string {
	var prev []byte

	for _, tx := range txs {
		h := sha256.New()                 //--- check if already exists
		h.Write(prev)                     // previous hash
		h.Write([]byte(tx.TransactionId)) // current tx id
		prev = h.Sum(nil)
	}

	return hex.EncodeToString(prev)
}

// manage trans-tokens, store trans-tokens with status 20 in Tokens table only if they are being pledged by the quorum currently
func (c *Core) getTxIdsQuorumIsPledgingFor(transTokensList []wallet.Token, userDID string) ([]TxnEpoch, error) {
	txnList := make([]TxnEpoch, 0)
	removeTransTokensList := make([]wallet.Token, 0)

	for _, transToken := range transTokensList {
		// TODO : 1. check latest block of each trans-token
		tokenType := RBTString
		if transToken.TokenValue < 1.0 {
			tokenType = PartString
		}
		latestBlock := c.w.GetLatestTokenBlock(transToken.TokenID, c.TokenType(tokenType))

		//		2. get txn-id
		txnId := latestBlock.GetTid()
		// if transaction-id is empty in latest block, then remove the trans-token from TokensTable
		if txnId == "" {
			removeTransTokensList = append(removeTransTokensList, transToken)
			continue
		}
		//    	3. check if txn-id is there in the TokenStateHash table
		tokenStateHashListByTxId, err := c.w.GetTokenStateHashByTransactionID(txnId)
		if err != nil {
			errMsg := fmt.Sprintf("failed to read TokenStateHash table, err : %v", err)
			return nil, fmt.Errorf("%v",errMsg)
		}
		// 		4. If it is not there remove the token from TokensTable
		if tokenStateHashListByTxId == nil {
			removeTransTokensList = append(removeTransTokensList, transToken)
			continue
		}
		// txn id found in latest block and in TokenStateHash table,
		// confirm it exists in TokensTable with status 20
		err = c.w.ReadTransTokenWithTokenIdAndDID(transToken.TokenID, userDID)
		if err != nil {
			// add it to tokens table if it doesn't exist already
			transToken.DID = userDID
			transToken.TokenStatus = wallet.QuorumPledgedForThisToken
			transToken.TransactionID = txnId
			err = c.w.AddTransTokenBeingPledged(transToken)
			if err != nil {
				// DO NOT RETURN ERROR, continue adding txnId
				errMsg := fmt.Sprintf("failed to write trans-tokens to TokensTable, err : %v", err)
				c.log.Error(errMsg)
			}
		}
		// add it to txn list to commit
		txnList = append(txnList, TxnEpoch{
			TransactionId: txnId,
			TxnEpoch:      latestBlock.GetEpoch(),
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

package wallet

import (
	"fmt"
	"strings"

	"github.com/rubixchain/rubixgoplatform/block"
	ut "github.com/rubixchain/rubixgoplatform/util"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"

	tkn "github.com/rubixchain/rubixgoplatform/token"
)

func tcsKeyDID(tcsKey string, did string) string {
	if strings.HasPrefix(tcsKey, did+"-") {
		return tcsKey
	}
	return did + "-" + tcsKey
}

func tcsKeyBlkNum(tokenType int, t string, blockNum uint64) string {
	tt := "wt"
	switch tokenType {
	case tkn.RBTTokenType:
		tt = WholeTokenType
	case tkn.PartTokenType:
		tt = PartTokenType
	case tkn.TestPartTokenType:
		tt = TestPartTokenType
	case tkn.NFTTokenType:
		tt = NFTType
	case tkn.TestNFTTokenType:
		tt = TestNFTType
	case tkn.TestTokenType:
		tt = TestTokenType
	case tkn.SmartContractTokenType:
		tt = SmartContractTokenType
	case tkn.FTTokenType:
		tt = FTTokenType
	}

	return tt + "-" + t + "-" + fmt.Sprintf("%016x", blockNum)

}

// getAllFullNodeBlocks gets the chain blocks from the FullNode storage
func (w *Wallet) getAllFullNodeBlocks(tt int, token string, blockID string, prefixDID string) ([][]byte, string, error) {
	db := w.fullNodeStorage
	if db == nil {
		return nil, "", fmt.Errorf("failed get all blocks, invalid token type")
	}
	iter := db.NewIterator(util.BytesPrefix([]byte(tcsPrefix(tt, token, prefixDID))), nil)
	defer iter.Release()
	blks := make([][]byte, 0)
	count := 0
	if blockID != "" {
		if !iter.Seek([]byte(tcsKey(tt, token, blockID))) {
			return nil, "", fmt.Errorf("Token chain block does not exist")
		}
	}
	nextBlkID := ""
	var err error
	for iter.Next() {
		key := string(iter.Key())
		if isOldKey(key) {
			err = w.updateFullNodeNewKey(tt, token, prefixDID)
			if err != nil {
				w.log.Error("Failed to update new key", "err", err)
				return nil, "", err
			}
			return w.getAllFullNodeBlocks(tt, token, blockID, prefixDID)
		}
		v := iter.Value()
		blk := make([]byte, len(v))
		copy(blk, v)
		if string(blk[0:2]) == ReferenceType {
			blk, err = w.getRawBlock(db, blk)
			if err != nil {
				return nil, "", err
			}
		}
		blks = append(blks, blk)
		count++
		if count == TCBlockCountLimit {
			b := block.InitBlock(blk, nil)
			blkID, err := b.GetBlockID(token)
			if err != nil {
				return nil, "", fmt.Errorf("invalid token chain block")
			}
			nextBlkID = blkID
		}
	}
	return blks, nextBlkID, nil
}

func (w *Wallet) addFullNodeMissingBlock(token string, b *block.Block, prefixDID string) error {
	opt := &opt.WriteOptions{
		Sync: true,
	}
	tt := b.GetTokenType(token)
	var db *ChainDB
	if w.IsFullNode {
		db = w.fullNodeStorage
	} else {
		w.log.Error("Not a fullnode, fullnode storage does not exist")
		return fmt.Errorf("not a fullnode, fullnode storage does not exist")
	}

	bid, err := b.GetBlockID(token)
	if err != nil {
		return err
	}
	key := tcsKey(tt, token, bid)
	if prefixDID != "" {
		key = tcsKeyDID(key, prefixDID)
	}
	lb := w.getFullNodeLatestBlock(tt, token, prefixDID)
	bn, err := b.GetBlockNumber(token)
	if err != nil {
		w.log.Error("Failed to get block number", "err", err)
		return err
	}

	// First block check block number start with zero
	if lb == nil {
		if bn != 0 {
			w.log.Error("Invalid block number, expect 0", "bn", bn)
			return fmt.Errorf("invalid block number")
		}
	} else {
		lbn, err := lb.GetBlockNumber(token)
		if err != nil {
			w.log.Error("Failed to get block number", "err", err)
			return err
		}
		if bn > lbn {
			// This might be a new block that arrived while we were syncing
			// Try to add it as a regular block instead
			w.log.Warn("Block number higher than latest, attempting regular add", "lbn", lbn, "bn", bn)
			return w.addFullNodeBlock(token, b, prefixDID)
		}
	}
	if b.CheckMultiTokenBlock() {
		bs, err := b.GetHash()
		if err != nil {
			return err
		}
		hs := ut.HexToStr(ut.CalculateHash(b.GetBlock(), "SHA3-256"))
		refkey := []byte(ReferenceType + "-" + hs + "-" + bs)
		_, err = w.getRawBlock(db, refkey)
		// Write only if reference block not exist
		if err != nil {
			db.l.Lock()
			err = db.Put(refkey, b.GetBlock(), opt)
			db.l.Unlock()
			if err != nil {
				return err
			}
		}
		db.l.Lock()
		err = db.Put([]byte(key), refkey, opt)
		db.l.Unlock()
		return err
	} else {
		db.l.Lock()
		err = db.Put([]byte(key), b.GetBlock(), opt)
		if tt == tkn.TestTokenType {
			w.log.Debug("Written", "key", key)
		}
		db.l.Unlock()
		return err
	}
}

func (w *Wallet) CopyFullNodeTokenChainToDIDPrefix(tt int, token string, prefixDID string) error {

	if prefixDID == "" {
		return fmt.Errorf("prefixDID cannot be empty")
	}

	db := w.fullNodeStorage
	if db == nil {
		return fmt.Errorf("fullnode storage is nil")
	}

	// Iterate ONLY over old-style keys (no DID prefix)
	iter := db.NewIterator(
		util.BytesPrefix([]byte(tcsPrefix(tt, token, ""))),
		nil,
	)
	defer iter.Release()

	for iter.Next() {
		oldKey := string(iter.Key())

		// Skip if this key already has DID prefix
		if strings.HasPrefix(oldKey, prefixDID+"-") {
			continue
		}

		// Construct new DID-based key
		newKey := tcsKeyDID(oldKey, prefixDID)

		// If DID-key already exists → skip
		exists, err := db.Has([]byte(newKey), nil)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		// Copy value safely
		v := iter.Value()
		blk := make([]byte, len(v))
		copy(blk, v)

		// Write DID-key without touching old key
		db.l.Lock()
		err = db.Put([]byte(newKey), blk, nil)
		db.l.Unlock()
		if err != nil {
			return err
		}
	}

	if err := iter.Error(); err != nil {
		return err
	}

	return nil
}

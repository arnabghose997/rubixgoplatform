package wallet

import (
	"fmt"
	"math"

	"github.com/rubixchain/rubixgoplatform/token"
)

type NewTokenRange struct {
	LowerBound int `json:"lower_bound"`
	UpperBound int `json:"upper_bound"`
}

// assign user level no. and range of token numbers
func (w *Wallet) AssignNewTokensToUser(userDID string) error {
	// lock fullnode wallet till the new tokens assignment for current user completes
	w.l.Lock()
	defer w.l.Unlock()

	// get the last entry Id of the NewTokensTable of fullnode
	newTokensLatestId := w.GetNewTokensTableLatestId()
	// get the latest level number and token number which is distributed
	latestIdDetails, err := w.ReadNewTokensBySlNum(newTokensLatestId)
	latestLevelNum := latestIdDetails.Level
	latestTokenNum := latestIdDetails.RangeUpperBound

	// TODO : lock DB of fullnode such that no other node can be assigned new tokens at this time.
	// Also maintain a queue for the users

	// assign new tokens to user with status 'free'
	latestLevelNum, latestTokenNum, err = w.AssignNewTokensByStatus(latestLevelNum, latestTokenNum, TokenIsFree, userDID)
	if err != nil {
		return err
	}

	// assign new tokens to user with status 'locked'
	latestLevelNum, latestTokenNum, err = w.AssignNewTokensByStatus(latestLevelNum, latestTokenNum, TokenIsLocked, userDID)
	if err != nil {
		return err
	}

	// assign new tokens to user with status 'pledged'
	latestLevelNum, latestTokenNum, err = w.AssignNewTokensByStatus(latestLevelNum, latestTokenNum, TokenIsPledged, userDID)
	if err != nil {
		return err
	}

	// assign new tokens to user with status 'committed'
	latestLevelNum, latestTokenNum, err = w.AssignNewTokensByStatus(latestLevelNum, latestTokenNum, TokenIsCommitted, userDID)
	if err != nil {
		return err
	}

	return nil
}

// read tokens of the user by given token status, get the total amount, then assign the user new tokens level and range as per the total amount
func (w *Wallet) AssignNewTokensByStatus(latestLevel, latestTokenNumber, tokenStatus int, userDID string) (int, int, error) {
	// read sqlite db to get all tokens owned by the userDID with given token status
	rbtList, err := w.ReadUsersRBTByStatus(userDID, tokenStatus)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get user rbts with status : %d, user :%s ; err: %v", tokenStatus, userDID, err)
		w.log.Error(errMsg)
		return -1, -1, fmt.Errorf("%v", errMsg)
	}

	totalRbt := 0.0

	for _, rbt := range rbtList {
		totalRbt += rbt.TokenValue
	}
	w.log.Debug("total rbt count ", len(rbtList))
	w.log.Debug("total rbt ", totalRbt)

	// assign required free RBTs to user
	userNewToken := NewTokensCount{
		DID:           userDID,
		PendingAmount: totalRbt - math.Floor(totalRbt),
		TokenStatus:   tokenStatus,
	}

	userNewTokensRange, err := w.CalculateNewTokensRange(int(math.Floor(totalRbt)), latestLevel, latestTokenNumber)
	if err != nil {
		errMsg := fmt.Sprintf("failed to assign new tokens range for user : %s; err : %v", userDID, err)
		return -1, -1, fmt.Errorf("%v", errMsg)
	}

	for assignedLevel, assignedRange := range userNewTokensRange {
		userNewToken.Level = assignedLevel
		userNewToken.RangeLowerBound = assignedRange.LowerBound
		userNewToken.RangeUpperBound = assignedRange.UpperBound

		// add new token assignment details to db
		err := w.AddNewTokenAssignment(userNewToken)
		if err != nil {
			errMsg := fmt.Sprintf("failed to add new tokens range to db for user : %s; err : %v", userDID, err)
			return -1, -1, fmt.Errorf("%v", errMsg)
		}
	}

	updatedLevel := userNewToken.Level
	updatedTokenNumber := userNewToken.RangeUpperBound
	return updatedLevel, updatedTokenNumber, nil
}

func (w *Wallet) CalculateNewTokensRange(requiredTokensCount, latestLevel, latestTokenNumber int) (map[int]*NewTokenRange, error) {
	newTokenAssignmentMap := make(map[int]*NewTokenRange, 0)
	var assignedLevel int
	var err error

	// check if token number limit is reached for a level
	if token.TokenMap[latestLevel] < latestTokenNumber { // if the latest token number exceeds the current level, then the assignment was invalid and we cannot move further
		errMsg := fmt.Sprintf("latest token number : %d exceeds the tokens-limit : %d of the latest level : %d", latestTokenNumber, token.TokenMap[latestLevel], latestLevel)
		w.log.Error(errMsg)
		return newTokenAssignmentMap, fmt.Errorf("%v", errMsg)
	} else if token.TokenMap[latestLevel] == latestTokenNumber { // if the limit of the current level is reached, start with next level
		assignedLevel = latestLevel + 1
		newTokenAssignmentMap[assignedLevel] = &NewTokenRange{
			LowerBound: 1,
			UpperBound: requiredTokensCount,
		}
	} else { // if the limit of the level is not reached, then continue with the current level
		assignedLevel = latestLevel
		newTokenAssignmentMap[assignedLevel] = &NewTokenRange{
			LowerBound: latestTokenNumber + 1,
			UpperBound: latestTokenNumber + requiredTokensCount,
		}
	}

	// if the upper bound of the assigned token range exceeds the token limit of the assigned level, then assign multiple levels with multiple ranges
	if newTokenAssignmentMap[assignedLevel].UpperBound > token.TokenMap[assignedLevel] {
		newTokenAssignmentMap[assignedLevel].UpperBound = token.TokenMap[assignedLevel]
		requiredTokensCount = requiredTokensCount - newTokenAssignmentMap[assignedLevel].UpperBound
		newTokenAssignmentMap, err = w.CalculateNewTokensRange(requiredTokensCount, assignedLevel, newTokenAssignmentMap[assignedLevel].UpperBound)
		if err != nil {
			return nil, err
		}
	}
	return newTokenAssignmentMap, nil
}

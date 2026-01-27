package wallet

import (
	"fmt"

	"github.com/rubixchain/rubixgoplatform/token"
)

type NewTokenRange struct {
	Level      int `json:"level"`
	LowerBound int `json:"lower_bound"`
	UpperBound int `json:"upper_bound"`
}

const (
	FreeToken      string = "free"
	LockedToken    string = "locked"
	PledgedToken   string = "pledged"
	CommittedToken string = "committed"
)

// // assign user level no. and range of token numbers
// func (w *Wallet) AssignNewTokensToUser(userDID string, userTotalRbt int) (map[int][]NewTokenRange, error) {
// 	// lock fullnode wallet till the new tokens assignment for current user completes
// 	w.l.Lock()
// 	defer w.l.Unlock()
//
// 	// get the last entry Id of the NewTokensTable of fullnode
// 	newTokensLatestId := w.GetNewTokensTableLatestId()
// 	// get the latest level number and token number which is distributed
// 	latestIdDetails, err := w.ReadNewTokensBySlNum(newTokensLatestId)
// 	latestLevelNum := latestIdDetails.Level
// 	latestTokenNum := latestIdDetails.RangeUpperBound
//
// 	// assign new tokens to user with status 'free'
// 	latestLevelNum, latestTokenNum, err = w.AssignNewTokensByStatus(latestLevelNum, latestTokenNum, TokenIsFree, userDID)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	// assign new tokens to user with status 'locked'
// 	latestLevelNum, latestTokenNum, err = w.AssignNewTokensByStatus(latestLevelNum, latestTokenNum, TokenIsLocked, userDID)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	// assign new tokens to user with status 'pledged'
// 	latestLevelNum, latestTokenNum, err = w.AssignNewTokensByStatus(latestLevelNum, latestTokenNum, TokenIsPledged, userDID)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	// assign new tokens to user with status 'committed'
// 	latestLevelNum, latestTokenNum, err = w.AssignNewTokensByStatus(latestLevelNum, latestTokenNum, TokenIsCommitted, userDID)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	// gather user's new tokens details
// 	newTokensMap, err := w.GatherUserNewTokens(userDID)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	return newTokensMap, nil
// }

// // read tokens of the user by given token status, get the total amount, then assign the user new tokens level and range as per the total amount
// func (w *Wallet) AssignNewTokensByStatus(latestLevel, latestTokenNumber, tokenStatus int, userDID string) (int, int, error) {
// 	// read sqlite db to get all tokens owned by the userDID with given token status
// 	rbtList, err := w.ReadUsersRBTByStatus(userDID, tokenStatus)
// 	if err != nil {
// 		errMsg := fmt.Sprintf("failed to get user rbts with status : %d, user :%s ; err: %v", tokenStatus, userDID, err)
// 		w.log.Error(errMsg)
// 		return -1, -1, fmt.Errorf("%v", errMsg)
// 	}
//
// 	totalRbtFlt := 0.0
//
// 	for _, rbt := range rbtList {
// 		totalRbtFlt += rbt.TokenValue
// 	}
//
// 	// assign the next whole amount of
// 	totalRbt := math.Ceil(totalRbtFlt)
// 	w.log.Debug("total rbt count ", len(rbtList))
// 	w.log.Debug("total rbt ", totalRbt)
//
// 	// assign required free RBTs to user
// 	userNewToken := NewTokensCount{
// 		DID: userDID,
// 		// PendingAmount: totalRbt - math.Ceil(totalRbt),
// 		TokenStatus: tokenStatus,
// 	}
//
// 	userNewTokensRange, err := w.CalculateNewTokensRange(int(totalRbt), latestLevel, latestTokenNumber)
// 	if err != nil {
// 		errMsg := fmt.Sprintf("failed to assign new tokens range for user : %s; err : %v", userDID, err)
// 		return -1, -1, fmt.Errorf("%v", errMsg)
// 	}
//
// 	for assignedLevel, assignedRange := range userNewTokensRange {
// 		userNewToken.Level = assignedLevel
// 		userNewToken.RangeLowerBound = assignedRange.LowerBound
// 		userNewToken.RangeUpperBound = assignedRange.UpperBound
//
// 		// add new token assignment details to db
// 		err := w.AddNewTokenAssignment(userNewToken)
// 		if err != nil {
// 			errMsg := fmt.Sprintf("failed to add new tokens range to db for user : %s; err : %v", userDID, err)
// 			return -1, -1, fmt.Errorf("%v", errMsg)
// 		}
// 	}
//
// 	updatedLevel := userNewToken.Level
// 	updatedTokenNumber := userNewToken.RangeUpperBound
// 	return updatedLevel, updatedTokenNumber, nil
// }

// assign the user new tokens level and range as per the total amount
func (w *Wallet) AssignNewTokensToUser(userDID string, totalAmount int) ([]NewTokensCount, error) {
	// lock fullnode wallet till the new tokens assignment for current user completes
	w.l.Lock()
	defer w.l.Unlock()

	// get the last entry Id of the NewTokensTable of fullnode
	newTokensLatestId := w.GetNewTokensTableLatestId()
	// get the latest level number and token number which is distributed
	latestIdDetails, err := w.ReadNewTokensBySlNum(newTokensLatestId)
	latestLevel := latestIdDetails.Level
	latestTokenNumber := latestIdDetails.RangeUpperBound

	// assign required free RBTs to user
	userNewToken := NewTokensCount{
		DID: userDID,
	}
	userNewTokensList := make([]NewTokensCount, 0)

	userNewTokensRange, err := w.CalculateNewTokensRange(totalAmount, latestLevel, latestTokenNumber)
	if err != nil {
		errMsg := fmt.Sprintf("failed to assign new tokens range for user : %s; err : %v", userDID, err)
		return nil, fmt.Errorf("%v", errMsg)
	}

	for assignedLevel, assignedRange := range userNewTokensRange {
		userNewToken.Level = assignedLevel
		userNewToken.RangeLowerBound = assignedRange.LowerBound
		userNewToken.RangeUpperBound = assignedRange.UpperBound

		// add new token assignment details to db
		err := w.AddNewTokenAssignment(userNewToken)
		if err != nil {
			errMsg := fmt.Sprintf("failed to add new tokens range to db for user : %s; err : %v", userDID, err)
			return nil, fmt.Errorf("%v", errMsg)
		}
		userNewTokensList = append(userNewTokensList, userNewToken)
	}
	return userNewTokensList, nil
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

// func (w *Wallet) GatherUserNewTokens(userDID string) (map[int][]NewTokenRange, error) {
// 	newTokensMap := make(map[int][]NewTokenRange, 0)
// 	userNewTokens, err := w.ReadUsersNewTokensRange(userDID)
// 	if err != nil {
// 		errMsg := fmt.Sprintf("failed to read user's new tokens from table, err : %v", err)
// 		w.log.Error(errMsg)
// 		return nil, fmt.Errorf("%v", errMsg)
// 	}
//
// 	for _, newTokenRange := range userNewTokens {
// 		newTokenRange_ := NewTokenRange{
// 			Level:      newTokenRange.Level,
// 			LowerBound: newTokenRange.RangeLowerBound,
// 			UpperBound: newTokenRange.RangeUpperBound,
// 		}
// 		newTokensMap[newTokenRange.TokenStatus] = append(newTokensMap[newTokenRange.TokenStatus], newTokenRange_)
// 		// switch newTokenRange.TokenStatus {
// 		// case TokenIsFree:
// 		// 	newTokensMap[FreeToken] = append(newTokensMap[FreeToken], newTokenRange_)
// 		// case TokenIsLocked:
// 		// 	newTokensMap[LockedToken] = append(newTokensMap[LockedToken], newTokenRange_)
// 		// case TokenIsPledged:
// 		// 	newTokensMap[PledgedToken] = append(newTokensMap[PledgedToken], newTokenRange_)
// 		// case TokenIsCommitted:
// 		// 	newTokensMap[CommittedToken] = append(newTokensMap[CommittedToken], newTokenRange_)
// 		// default:
// 		// 	errMsg := fmt.Sprintf("invalid token status : %v, invalid assignment to user : %v", newTokenRange.TokenStatus, userDID)
// 		// 	w.log.Error(errMsg)
// 		// 	return nil, fmt.Errorf("%v", errMsg)
// 		// }
// 	}
// 	return newTokensMap, nil
// }

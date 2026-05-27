package handlers

import (
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"private-ethereum-voting/backend/models"
)

func (a *App) Vote(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r)
	defer cancel()

	var req models.VoteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON request")
		return
	}

	req.Voter = strings.TrimSpace(req.Voter)
	if req.Voter == "" {
		writeError(w, http.StatusBadRequest, "Voter is required")
		return
	}
	if req.CandidateID == 0 {
		writeError(w, http.StatusBadRequest, "Candidate does not exist")
		return
	}

	instance, err := a.contract()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	opts := &bind.CallOpts{Context: ctx}
	isActive, err := instance.IsActive(opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !isActive {
		writeError(w, http.StatusConflict, "Voting is not active")
		return
	}

	count, err := instance.GetCandidatesCount(opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.CandidateID > count.Uint64() {
		writeError(w, http.StatusBadRequest, "Candidate does not exist")
		return
	}

	auth, voterAddress, err := a.voterAuth(ctx, req.Voter)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hasVoted, err := instance.HasAddressVoted(opts, voterAddress)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if hasVoted {
		writeError(w, http.StatusConflict, "Already voted")
		return
	}

	// В этот момент backend отправляет настоящую Ethereum-транзакцию.
	// Отправитель транзакции определяется приватным ключом выбранного voter.
	tx, err := instance.Vote(auth, new(big.Int).SetUint64(req.CandidateID))
	if err != nil {
		writeError(w, http.StatusBadRequest, simplifyRevert(err))
		return
	}
	if _, err := a.waitTx(ctx, tx); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "Vote accepted", TxHash: tx.Hash().Hex()})
}

func (a *App) HasVoted(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r)
	defer cancel()

	voter := r.PathValue("voter")
	_, voterAddress, err := a.voterAuth(ctx, voter)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	instance, err := a.contract()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hasVoted, err := instance.HasAddressVoted(&bind.CallOpts{Context: ctx}, voterAddress)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"voter":    voter,
		"address":  voterAddress.Hex(),
		"hasVoted": hasVoted,
	})
}

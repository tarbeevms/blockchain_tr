package handlers

import (
	"net/http"

	"private-ethereum-voting/backend/models"
)

func (a *App) AddCandidate(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r)
	defer cancel()

	var req models.CandidateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON request")
		return
	}

	name := normalizeName(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "Candidate name is required")
		return
	}

	instance, err := a.contract()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	auth, err := a.adminAuth(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Добавление кандидата меняет storage контракта, поэтому backend вызывает
	// не eth_call, а отправляет транзакцию от admin-аккаунта.
	tx, err := instance.AddCandidate(auth, name)
	if err != nil {
		writeError(w, http.StatusBadRequest, simplifyRevert(err))
		return
	}
	if _, err := a.waitTx(ctx, tx); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "Candidate added", TxHash: tx.Hash().Hex()})
}

func (a *App) GetCandidates(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r)
	defer cancel()

	candidates, err := a.allCandidates(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, candidates)
}

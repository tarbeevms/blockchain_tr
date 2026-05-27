package handlers

import (
	"net/http"

	"private-ethereum-voting/backend/eth"
	"private-ethereum-voting/backend/models"
)

func (a *App) Deploy(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r)
	defer cancel()

	// Деплой выполняется от имени admin. Именно этот адрес станет admin
	// внутри Solidity constructor, потому что contract constructor видит
	// отправителя deploy-транзакции как msg.sender.
	auth, err := a.adminAuth(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// eth.DeployVoting формирует Ethereum deploy-транзакцию с bytecode
	// контракта Voting и отправляет ее в локальный Geth.
	address, tx, _, err := eth.DeployVoting(auth, a.client)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Backend ждет receipt, чтобы вернуть frontend-у не просто hash, а
	// подтверждение, что транзакция включена в блок и не reverted.
	if _, err := a.waitTx(ctx, tx); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Адрес хранится в памяти backend-а. После перезапуска backend можно либо
	// снова задеплоить контракт, либо прописать адрес в config.json/ENV.
	a.setContractAddress(address)
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, ContractAddress: address.Hex(), Message: "Contract deployed", TxHash: tx.Hash().Hex()})
}

func (a *App) StartVoting(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r)
	defer cancel()

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

	tx, err := instance.StartVoting(auth)
	if err != nil {
		writeError(w, http.StatusBadRequest, simplifyRevert(err))
		return
	}
	if _, err := a.waitTx(ctx, tx); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "Voting started", TxHash: tx.Hash().Hex()})
}

func (a *App) StopVoting(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r)
	defer cancel()

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

	tx, err := instance.StopVoting(auth)
	if err != nil {
		writeError(w, http.StatusBadRequest, simplifyRevert(err))
		return
	}
	if _, err := a.waitTx(ctx, tx); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "Voting stopped", TxHash: tx.Hash().Hex()})
}

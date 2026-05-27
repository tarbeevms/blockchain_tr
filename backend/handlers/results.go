package handlers

import (
	"net/http"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"private-ethereum-voting/backend/eth"
)

func (a *App) GetResults(w http.ResponseWriter, r *http.Request) {
	a.GetCandidates(w, r)
}

func (a *App) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r)
	defer cancel()

	a.mu.RLock()
	address := a.contractAddress
	a.mu.RUnlock()

	accounts := make(map[string]string)
	if adminAddress, err := eth.AddressFromPrivateKey(a.cfg.AdminPrivateKey); err == nil {
		accounts["admin"] = adminAddress.Hex()
	}
	for voter, privateKey := range a.cfg.Voters {
		if voterAddress, err := eth.AddressFromPrivateKey(privateKey); err == nil {
			accounts[voter] = voterAddress.Hex()
		}
	}

	response := map[string]interface{}{
		"success":         true,
		"deployed":        address != (common.Address{}),
		"contractAddress": "",
		"isActive":        false,
		"candidatesCount": uint64(0),
		"accounts":        accounts,
	}

	if address == (common.Address{}) {
		writeJSON(w, http.StatusOK, response)
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
	count, err := instance.GetCandidatesCount(opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response["contractAddress"] = address.Hex()
	response["isActive"] = isActive
	response["candidatesCount"] = count.Uint64()
	writeJSON(w, http.StatusOK, response)
}

func simplifyRevert(err error) string {
	message := err.Error()
	if message == "" {
		return "Ethereum transaction failed"
	}
	return message
}

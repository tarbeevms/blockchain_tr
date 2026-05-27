package handlers

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	votingcontract "private-ethereum-voting/backend/contract"
	"private-ethereum-voting/backend/eth"
	"private-ethereum-voting/backend/models"
)

const defaultExplorerLimit = 20
const maxExplorerLimit = 100
const maxExplorerScanDepth = 5000

func (a *App) GetBlockchain(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := requestContext(r)
	defer cancel()

	// Geth создает много пустых блоков, поэтому explorer сканирует цепочку
	// назад и возвращает именно блоки с транзакциями.
	limit := explorerLimit(r)
	header, err := a.client.HeaderByNumber(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	parsed, err := abi.JSON(strings.NewReader(votingcontract.VotingABI))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	latest := header.Number.Uint64()
	roles := a.accountRoles()
	blocks := make([]models.BlockView, 0, limit)
	discoveredContract := ""
	scanned := 0
	for number := latest; ; number-- {
		block, err := a.client.BlockByNumber(ctx, new(big.Int).SetUint64(number))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		view := models.BlockView{
			Number:           block.NumberU64(),
			Hash:             block.Hash().Hex(),
			ParentHash:       block.ParentHash().Hex(),
			Time:             block.Time(),
			Miner:            block.Coinbase().Hex(),
			TransactionCount: len(block.Transactions()),
			Transactions:     make([]models.TxView, 0, len(block.Transactions())),
		}

		for _, tx := range block.Transactions() {
			txView := a.explorerTx(ctx, &parsed, block, tx, roles)
			if txView.ContractCreated != "" && txView.Function == "constructor / deploy Voting" && discoveredContract == "" {
				discoveredContract = txView.ContractCreated
			}
			view.Transactions = append(view.Transactions, txView)
		}

		if view.TransactionCount > 0 {
			blocks = append(blocks, view)
		}
		scanned++
		if len(blocks) >= limit || number == 0 || scanned >= maxExplorerScanDepth {
			break
		}
	}

	a.mu.RLock()
	contractAddress := a.contractAddress.Hex()
	if a.contractAddress == (common.Address{}) {
		contractAddress = ""
	}
	a.mu.RUnlock()
	if contractAddress == "" {
		contractAddress = discoveredContract
	}

	writeJSON(w, http.StatusOK, models.BlockchainView{
		Success:         true,
		LatestBlock:     latest,
		ContractAddress: contractAddress,
		Accounts:        roleMapToStrings(roles),
		Blocks:          blocks,
	})
}

func (a *App) explorerTx(ctx context.Context, parsed *abi.ABI, block *types.Block, tx *types.Transaction, roles map[common.Address]string) models.TxView {
	view := models.TxView{
		Hash:        tx.Hash().Hex(),
		BlockNumber: block.NumberU64(),
		Nonce:       tx.Nonce(),
		ValueWei:    tx.Value().String(),
		Gas:         tx.Gas(),
		Status:      "unknown",
		Type:        "contract_call",
	}

	if tx.To() == nil {
		// У deploy-транзакции нет получателя: адрес контракта появляется
		// только после майнинга и хранится в receipt.ContractAddress.
		view.Type = "contract_deploy"
	} else {
		view.To = tx.To().Hex()
		view.ToRole = roles[*tx.To()]
	}

	signer := types.LatestSignerForChainID(a.chainID)
	if sender, err := types.Sender(signer, tx); err == nil {
		view.From = sender.Hex()
		view.FromRole = roles[sender]
	} else {
		view.Error = err.Error()
	}

	receipt, err := a.client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		view.Error = joinExplorerError(view.Error, err.Error())
	} else {
		view.GasUsed = receipt.GasUsed
		if receipt.Status == types.ReceiptStatusSuccessful {
			view.Status = "success"
		} else {
			view.Status = "reverted"
		}
		if receipt.ContractAddress != (common.Address{}) {
			view.ContractCreated = receipt.ContractAddress.Hex()
		}
		view.Events = decodeEvents(parsed, receipt.Logs)
	}

	view.Function, view.Arguments, view.InputSelector = decodeInput(parsed, tx.Data(), tx.To() == nil)
	return view
}

func decodeInput(parsed *abi.ABI, input []byte, isDeploy bool) (string, []models.TxArg, string) {
	if isDeploy {
		return "constructor / deploy Voting", nil, ""
	}
	if len(input) < 4 {
		return "plain ETH transfer", nil, ""
	}

	selector := "0x" + common.Bytes2Hex(input[:4])
	// Первые 4 байта calldata - selector функции. По ABI можно восстановить,
	// какая функция была вызвана и какие аргументы были переданы.
	method, err := parsed.MethodById(input[:4])
	if err != nil {
		return "unknown", nil, selector
	}

	values, err := method.Inputs.Unpack(input[4:])
	if err != nil {
		return method.Name, nil, selector
	}

	args := make([]models.TxArg, 0, len(values))
	for i, value := range values {
		inputDef := method.Inputs[i]
		args = append(args, models.TxArg{
			Name:  inputDef.Name,
			Type:  inputDef.Type.String(),
			Value: formatExplorerValue(value),
		})
	}

	return method.Name, args, selector
}

func decodeEvents(parsed *abi.ABI, logs []*types.Log) []models.EventView {
	events := make([]models.EventView, 0, len(logs))
	for _, item := range logs {
		if len(item.Topics) == 0 {
			continue
		}

		switch item.Topics[0] {
		case parsed.Events["CandidateAdded"].ID:
			// Indexed-поля события лежат в topics, остальные поля - в data.
			args := []models.TxArg{}
			if len(item.Topics) > 1 {
				args = append(args, models.TxArg{Name: "id", Type: "uint256", Value: new(big.Int).SetBytes(item.Topics[1].Bytes()).String()})
			}
			values := map[string]interface{}{}
			if err := parsed.UnpackIntoMap(values, "CandidateAdded", item.Data); err == nil {
				args = append(args, models.TxArg{Name: "name", Type: "string", Value: formatExplorerValue(values["name"])})
			}
			events = append(events, models.EventView{Name: "CandidateAdded", Address: item.Address.Hex(), Arguments: args})
		case parsed.Events["VotingStarted"].ID:
			events = append(events, models.EventView{Name: "VotingStarted", Address: item.Address.Hex()})
		case parsed.Events["VotingStopped"].ID:
			events = append(events, models.EventView{Name: "VotingStopped", Address: item.Address.Hex()})
		case parsed.Events["VoteCast"].ID:
			args := []models.TxArg{}
			if len(item.Topics) > 1 {
				args = append(args, models.TxArg{Name: "voter", Type: "address", Value: common.BytesToAddress(item.Topics[1].Bytes()[12:]).Hex()})
			}
			if len(item.Topics) > 2 {
				args = append(args, models.TxArg{Name: "candidateId", Type: "uint256", Value: new(big.Int).SetBytes(item.Topics[2].Bytes()).String()})
			}
			events = append(events, models.EventView{Name: "VoteCast", Address: item.Address.Hex(), Arguments: args})
		default:
			events = append(events, models.EventView{Name: "unknown", Address: item.Address.Hex()})
		}
	}
	return events
}

func (a *App) accountRoles() map[common.Address]string {
	roles := make(map[common.Address]string)
	if address, err := eth.AddressFromPrivateKey(a.cfg.AdminPrivateKey); err == nil {
		roles[address] = "admin"
	}
	for voter, privateKey := range a.cfg.Voters {
		if address, err := eth.AddressFromPrivateKey(privateKey); err == nil {
			roles[address] = voter
		}
	}
	a.mu.RLock()
	if a.contractAddress != (common.Address{}) {
		roles[a.contractAddress] = "Voting contract"
	}
	a.mu.RUnlock()
	return roles
}

func roleMapToStrings(roles map[common.Address]string) map[string]string {
	result := make(map[string]string, len(roles))
	for address, role := range roles {
		result[role] = address.Hex()
	}
	return result
}

func explorerLimit(r *http.Request) int {
	limit := defaultExplorerLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	if limit < 1 {
		return 1
	}
	if limit > maxExplorerLimit {
		return maxExplorerLimit
	}
	return limit
}

func formatExplorerValue(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case common.Address:
		return typed.Hex()
	case *big.Int:
		return typed.String()
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func joinExplorerError(existing string, next string) string {
	if existing == "" {
		return next
	}
	return existing + "; " + next
}

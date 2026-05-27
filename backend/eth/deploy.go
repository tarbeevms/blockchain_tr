package eth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	votingcontract "private-ethereum-voting/backend/contract"
)

func DeployVoting(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *votingcontract.Voting, error) {
	// Здесь eth-пакет не знает деталей ABI/bytecode. Он делегирует деплой
	// пакету contract, а сам только добавляет понятный текст ошибки.
	address, tx, instance, err := votingcontract.DeployVoting(auth, backend)
	if err != nil {
		return common.Address{}, nil, nil, fmt.Errorf("deploy voting contract: %w", err)
	}
	return address, tx, instance, nil
}

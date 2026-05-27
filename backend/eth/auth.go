package eth

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func NewTransactor(privateKeyHex string, chainID *big.Int) (*bind.TransactOpts, common.Address, error) {
	key, err := ParsePrivateKey(privateKeyHex)
	if err != nil {
		return nil, common.Address{}, err
	}

	// TransactOpts содержит приватный ключ и chainId. go-ethereum использует
	// его для подписи транзакций так, будто их отправил соответствующий адрес.
	auth, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("create transactor: %w", err)
	}

	// GasLimit здесь является верхней границей. Реальный расход показывается
	// в receipt как gasUsed и обычно намного меньше лимита.
	auth.GasLimit = 5_000_000
	auth.GasTipCap = big.NewInt(1)
	auth.GasFeeCap = big.NewInt(1_000_000_000)

	return auth, crypto.PubkeyToAddress(key.PublicKey), nil
}

func AddressFromPrivateKey(privateKeyHex string) (common.Address, error) {
	key, err := ParsePrivateKey(privateKeyHex)
	if err != nil {
		return common.Address{}, err
	}

	return crypto.PubkeyToAddress(key.PublicKey), nil
}

func ParsePrivateKey(privateKeyHex string) (*ecdsa.PrivateKey, error) {
	cleaned := strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	key, err := crypto.HexToECDSA(cleaned)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return key, nil
}

package contract

import (
	_ "embed"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Voting.bin - это скомпилированный bytecode Solidity-контракта.
// Директива go:embed встраивает его прямо в backend-бинарник при сборке Docker-образа.
// Поэтому backend может деплоить контракт без чтения отдельного файла во время запуска.
//
//go:embed Voting.bin
var votingBin string

// VotingABI описывает внешний интерфейс контракта: какие функции и события
// существуют, какие у них аргументы и типы. ABI нужен backend-у, чтобы:
// 1. упаковывать вызовы функций в calldata;
// 2. распаковывать ответы view-функций;
// 3. декодировать события из receipt logs.
const VotingABI = `[
  {"inputs":[],"stateMutability":"nonpayable","type":"constructor"},
  {"anonymous":false,"inputs":[{"indexed":true,"internalType":"uint256","name":"id","type":"uint256"},{"indexed":false,"internalType":"string","name":"name","type":"string"}],"name":"CandidateAdded","type":"event"},
  {"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"voter","type":"address"},{"indexed":true,"internalType":"uint256","name":"candidateId","type":"uint256"}],"name":"VoteCast","type":"event"},
  {"anonymous":false,"inputs":[],"name":"VotingStarted","type":"event"},
  {"anonymous":false,"inputs":[],"name":"VotingStopped","type":"event"},
  {"inputs":[{"internalType":"string","name":"name","type":"string"}],"name":"addCandidate","outputs":[],"stateMutability":"nonpayable","type":"function"},
  {"inputs":[],"name":"admin","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
  {"inputs":[{"internalType":"uint256","name":"","type":"uint256"}],"name":"candidates","outputs":[{"internalType":"uint256","name":"id","type":"uint256"},{"internalType":"string","name":"name","type":"string"},{"internalType":"uint256","name":"voteCount","type":"uint256"}],"stateMutability":"view","type":"function"},
  {"inputs":[],"name":"candidatesCount","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
  {"inputs":[{"internalType":"uint256","name":"candidateId","type":"uint256"}],"name":"getCandidate","outputs":[{"internalType":"uint256","name":"id","type":"uint256"},{"internalType":"string","name":"name","type":"string"},{"internalType":"uint256","name":"voteCount","type":"uint256"}],"stateMutability":"view","type":"function"},
  {"inputs":[],"name":"getCandidatesCount","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
  {"inputs":[{"internalType":"address","name":"voter","type":"address"}],"name":"hasAddressVoted","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
  {"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"hasVoted","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
  {"inputs":[],"name":"isActive","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
  {"inputs":[],"name":"startVoting","outputs":[],"stateMutability":"nonpayable","type":"function"},
  {"inputs":[],"name":"stopVoting","outputs":[],"stateMutability":"nonpayable","type":"function"},
  {"inputs":[{"internalType":"uint256","name":"candidateId","type":"uint256"}],"name":"vote","outputs":[],"stateMutability":"nonpayable","type":"function"}
]`

type Voting struct {
	// address - адрес уже развернутого контракта в локальной Ethereum-сети.
	address common.Address

	// BoundContract - универсальный объект go-ethereum для вызова функций
	// контракта по ABI. Он умеет делать и read-only call, и транзакции.
	contract *bind.BoundContract
}

type Candidate struct {
	ID        *big.Int
	Name      string
	VoteCount *big.Int
}

func DeployVoting(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Voting, error) {
	parsed, err := parsedABI()
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	bytecode := strings.TrimSpace(votingBin)
	if bytecode == "" {
		return common.Address{}, nil, nil, fmt.Errorf("Voting.bin is empty; compile contracts/Voting.sol first")
	}

	// DeployContract формирует deploy-транзакцию. У такой транзакции нет поля
	// "to", потому что контракт еще не существует. Адрес контракта появляется
	// после майнинга транзакции и вычисляется из адреса отправителя и nonce.
	address, tx, bound, err := bind.DeployContract(auth, parsed, common.FromHex(bytecode), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	return address, tx, &Voting{address: address, contract: bound}, nil
}

func NewVoting(address common.Address, backend bind.ContractBackend) (*Voting, error) {
	parsed, err := parsedABI()
	if err != nil {
		return nil, err
	}

	// NewBoundContract не деплоит новый контракт. Он "привязывает" Go-код
	// к уже существующему адресу контракта, чтобы backend мог вызывать функции.
	bound := bind.NewBoundContract(address, parsed, backend, backend, backend)
	return &Voting{address: address, contract: bound}, nil
}

func (v *Voting) Address() common.Address {
	return v.address
}

func (v *Voting) AddCandidate(auth *bind.TransactOpts, name string) (*types.Transaction, error) {
	// Transact создает подписанную Ethereum-транзакцию, потому что addCandidate
	// меняет состояние блокчейна.
	return v.contract.Transact(auth, "addCandidate", name)
}

func (v *Voting) StartVoting(auth *bind.TransactOpts) (*types.Transaction, error) {
	// startVoting меняет bool isActive, поэтому это тоже транзакция.
	return v.contract.Transact(auth, "startVoting")
}

func (v *Voting) StopVoting(auth *bind.TransactOpts) (*types.Transaction, error) {
	return v.contract.Transact(auth, "stopVoting")
}

func (v *Voting) Vote(auth *bind.TransactOpts, candidateID *big.Int) (*types.Transaction, error) {
	// vote меняет mapping hasVoted и счетчик voteCount. Это запись в blockchain
	// state, поэтому требуется транзакция, gas и подпись приватным ключом voter.
	return v.contract.Transact(auth, "vote", candidateID)
}

func (v *Voting) IsActive(opts *bind.CallOpts) (bool, error) {
	// Call не создает транзакцию и не тратит gas, потому что isActive только
	// читает состояние контракта.
	out, err := v.call(opts, "isActive")
	if err != nil {
		return false, err
	}
	return asBool(out[0])
}

func (v *Voting) GetCandidatesCount(opts *bind.CallOpts) (*big.Int, error) {
	out, err := v.call(opts, "getCandidatesCount")
	if err != nil {
		return nil, err
	}
	return asBigInt(out[0])
}

func (v *Voting) GetCandidate(opts *bind.CallOpts, candidateID *big.Int) (Candidate, error) {
	out, err := v.call(opts, "getCandidate", candidateID)
	if err != nil {
		return Candidate{}, err
	}

	id, err := asBigInt(out[0])
	if err != nil {
		return Candidate{}, err
	}
	name, ok := out[1].(string)
	if !ok {
		return Candidate{}, fmt.Errorf("unexpected candidate name type %T", out[1])
	}
	voteCount, err := asBigInt(out[2])
	if err != nil {
		return Candidate{}, err
	}

	return Candidate{ID: id, Name: name, VoteCount: voteCount}, nil
}

func (v *Voting) HasAddressVoted(opts *bind.CallOpts, voter common.Address) (bool, error) {
	out, err := v.call(opts, "hasAddressVoted", voter)
	if err != nil {
		return false, err
	}
	return asBool(out[0])
}

func (v *Voting) call(opts *bind.CallOpts, method string, params ...interface{}) ([]interface{}, error) {
	var out []interface{}
	// BoundContract.Call выполняет eth_call через Geth. Это локальная симуляция
	// чтения состояния на выбранном блоке, без записи транзакции в блокчейн.
	if err := v.contract.Call(opts, &out, method, params...); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s returned no values", method)
	}
	return out, nil
}

func parsedABI() (abi.ABI, error) {
	// abi.JSON превращает JSON-описание ABI в структуру go-ethereum, с которой
	// можно кодировать calldata и декодировать результаты.
	parsed, err := abi.JSON(strings.NewReader(VotingABI))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("parse voting ABI: %w", err)
	}
	return parsed, nil
}

func asBigInt(value interface{}) (*big.Int, error) {
	switch typed := value.(type) {
	case *big.Int:
		return new(big.Int).Set(typed), nil
	case big.Int:
		return new(big.Int).Set(&typed), nil
	default:
		return nil, fmt.Errorf("unexpected uint256 type %T", value)
	}
}

func asBool(value interface{}) (bool, error) {
	typed, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("unexpected bool type %T", value)
	}
	return typed, nil
}

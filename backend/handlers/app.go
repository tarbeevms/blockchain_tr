package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"private-ethereum-voting/backend/config"
	votingcontract "private-ethereum-voting/backend/contract"
	"private-ethereum-voting/backend/eth"
	"private-ethereum-voting/backend/models"
)

type App struct {
	cfg             *config.Config
	client          *ethclient.Client
	chainID         *big.Int
	contractAddress common.Address
	mu              sync.RWMutex
}

func NewApp(cfg *config.Config, client *ethclient.Client) *App {
	app := &App{
		cfg:     cfg,
		client:  client,
		chainID: big.NewInt(cfg.ChainID),
	}
	if common.IsHexAddress(cfg.ContractAddress) {
		app.contractAddress = common.HexToAddress(cfg.ContractAddress)
	}
	return app
}

func (a *App) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/deploy", a.Deploy)
	mux.HandleFunc("POST /api/candidates", a.AddCandidate)
	mux.HandleFunc("GET /api/candidates", a.GetCandidates)
	mux.HandleFunc("POST /api/start", a.StartVoting)
	mux.HandleFunc("POST /api/stop", a.StopVoting)
	mux.HandleFunc("POST /api/vote", a.Vote)
	mux.HandleFunc("GET /api/results", a.GetResults)
	mux.HandleFunc("GET /api/status", a.GetStatus)
	mux.HandleFunc("GET /api/has-voted/{voter}", a.HasVoted)
	mux.HandleFunc("GET /api/blockchain", a.GetBlockchain)
}

func (a *App) contract() (*votingcontract.Voting, error) {
	a.mu.RLock()
	address := a.contractAddress
	a.mu.RUnlock()

	if address == (common.Address{}) {
		return nil, errors.New("contract is not deployed")
	}

	return votingcontract.NewVoting(address, a.client)
}

func (a *App) setContractAddress(address common.Address) {
	a.mu.Lock()
	a.contractAddress = address
	a.mu.Unlock()
}

func (a *App) adminAuth(ctx context.Context) (*bind.TransactOpts, error) {
	auth, _, err := eth.NewTransactor(a.cfg.AdminPrivateKey, a.chainID)
	if err != nil {
		return nil, err
	}
	auth.Context = ctx
	return auth, nil
}

func (a *App) voterAuth(ctx context.Context, voter string) (*bind.TransactOpts, common.Address, error) {
	privateKey, ok := a.cfg.Voters[voter]
	if !ok {
		return nil, common.Address{}, fmt.Errorf("unknown voter %q", voter)
	}

	auth, address, err := eth.NewTransactor(privateKey, a.chainID)
	if err != nil {
		return nil, common.Address{}, err
	}
	auth.Context = ctx

	return auth, address, nil
}

func (a *App) waitTx(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	receipt, err := bind.WaitMined(ctx, a.client, tx)
	if err != nil {
		return nil, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return receipt, fmt.Errorf("transaction %s reverted", tx.Hash().Hex())
	}
	return receipt, nil
}

func (a *App) allCandidates(ctx context.Context) ([]models.Candidate, error) {
	instance, err := a.contract()
	if err != nil {
		return nil, err
	}

	opts := &bind.CallOpts{Context: ctx}
	count, err := instance.GetCandidatesCount(opts)
	if err != nil {
		return nil, err
	}

	candidates := make([]models.Candidate, 0, count.Uint64())
	for i := uint64(1); i <= count.Uint64(); i++ {
		candidate, err := instance.GetCandidate(opts, new(big.Int).SetUint64(i))
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, models.Candidate{
			ID:        candidate.ID.Uint64(),
			Name:      candidate.Name,
			VoteCount: candidate.VoteCount.Uint64(),
		})
	}

	return candidates, nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.APIResponse{Success: false, Message: message})
}

func decodeJSON(r *http.Request, target interface{}) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func requestContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 45*time.Second)
}

func normalizeName(value string) string {
	return strings.TrimSpace(value)
}

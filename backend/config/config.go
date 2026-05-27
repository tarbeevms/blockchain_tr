package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// RPCURL - HTTP-адрес Geth JSON-RPC. Внутри Docker Compose backend
	// обращается к geth по имени сервиса: http://geth:8545.
	RPCURL string `json:"rpcUrl"`

	// ChainID защищает транзакции от повторного использования в другой сети.
	// В genesis.json нашей приватной сети указан такой же chainId: 2025.
	ChainID int64 `json:"chainId"`

	// ContractAddress можно задать, если контракт уже был развернут раньше.
	// Если поле пустое, адрес появляется после POST /api/deploy и хранится
	// в памяти backend-процесса до его перезапуска.
	ContractAddress string `json:"contractAddress"`

	// Приватный ключ admin используется backend-ом для деплоя контракта и
	// административных транзакций: addCandidate/startVoting/stopVoting.
	AdminPrivateKey string `json:"adminPrivateKey"`

	// Voters хранит приватные ключи учебных аккаунтов voter1/voter2/voter3.
	// В production так делать нельзя; здесь это сделано только для локальной
	// демонстрации без MetaMask и внешних кошельков.
	Voters map[string]string `json:"voters"`
}

func Load() (*Config, error) {
	// CONFIG_PATH позволяет заменить файл настроек без изменения кода.
	// По умолчанию backend читает backend/config.json.
	path := getenv("CONFIG_PATH", "config.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	// Значения из переменных окружения имеют приоритет над config.json.
	// Это удобно для Docker Compose: один и тот же образ backend можно запускать
	// с разными RPC URL, chainId или ключами.
	if value := os.Getenv("ETH_RPC_URL"); value != "" {
		cfg.RPCURL = value
	}
	if value := os.Getenv("CHAIN_ID"); value != "" {
		chainID, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse CHAIN_ID: %w", err)
		}
		cfg.ChainID = chainID
	}
	if value := os.Getenv("CONTRACT_ADDRESS"); value != "" {
		cfg.ContractAddress = value
	}
	if value := os.Getenv("ADMIN_PRIVATE_KEY"); value != "" {
		cfg.AdminPrivateKey = value
	}

	if cfg.Voters == nil {
		cfg.Voters = make(map[string]string)
	}
	for _, voter := range []string{"voter1", "voter2", "voter3"} {
		envName := "VOTER_" + voter[len("voter"):] + "_PRIVATE_KEY"
		if value := os.Getenv(envName); value != "" {
			cfg.Voters[voter] = value
		}
	}

	if cfg.RPCURL == "" {
		return nil, fmt.Errorf("rpcUrl is required")
	}
	if cfg.ChainID == 0 {
		return nil, fmt.Errorf("chainId is required")
	}
	if cfg.AdminPrivateKey == "" {
		return nil, fmt.Errorf("adminPrivateKey is required")
	}

	return &cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

package eth

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

func DialWithRetry(rpcURL string, attempts int, delay time.Duration) (*ethclient.Client, error) {
	var lastErr error

	for i := 1; i <= attempts; i++ {
		// ethclient.Dial создает Go-клиент к Ethereum JSON-RPC.
		// Дальше через этот объект backend вызывает методы Geth:
		// eth_getBlockByNumber, eth_sendRawTransaction, eth_call и другие.
		client, err := ethclient.Dial(rpcURL)
		if err == nil {
			return client, nil
		}

		// При старте через Docker Compose Geth может еще поднимать HTTP RPC,
		// поэтому backend не падает сразу, а пробует подключиться несколько раз.
		lastErr = err
		time.Sleep(delay)
	}

	return nil, fmt.Errorf("connect to Ethereum RPC %s: %w", rpcURL, lastErr)
}

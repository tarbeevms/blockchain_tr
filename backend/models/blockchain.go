package models

type BlockchainView struct {
	Success         bool              `json:"success"`
	LatestBlock     uint64            `json:"latestBlock"`
	ContractAddress string            `json:"contractAddress"`
	Accounts        map[string]string `json:"accounts"`
	Blocks          []BlockView       `json:"blocks"`
}

type BlockView struct {
	Number           uint64   `json:"number"`
	Hash             string   `json:"hash"`
	ParentHash       string   `json:"parentHash"`
	Time             uint64   `json:"time"`
	Miner            string   `json:"miner"`
	TransactionCount int      `json:"transactionCount"`
	Transactions     []TxView `json:"transactions"`
}

type TxView struct {
	Hash            string      `json:"hash"`
	BlockNumber     uint64      `json:"blockNumber"`
	From            string      `json:"from"`
	FromRole        string      `json:"fromRole,omitempty"`
	To              string      `json:"to,omitempty"`
	ToRole          string      `json:"toRole,omitempty"`
	Nonce           uint64      `json:"nonce"`
	ValueWei        string      `json:"valueWei"`
	Gas             uint64      `json:"gas"`
	GasUsed         uint64      `json:"gasUsed"`
	Status          string      `json:"status"`
	Type            string      `json:"type"`
	Function        string      `json:"function,omitempty"`
	Arguments       []TxArg     `json:"arguments,omitempty"`
	ContractCreated string      `json:"contractCreated,omitempty"`
	Events          []EventView `json:"events,omitempty"`
	InputSelector   string      `json:"inputSelector,omitempty"`
	Error           string      `json:"error,omitempty"`
}

type TxArg struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type EventView struct {
	Name      string  `json:"name"`
	Address   string  `json:"address"`
	Arguments []TxArg `json:"arguments,omitempty"`
}

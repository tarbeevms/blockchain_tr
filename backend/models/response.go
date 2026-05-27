package models

type APIResponse struct {
	Success         bool   `json:"success"`
	Message         string `json:"message,omitempty"`
	ContractAddress string `json:"contractAddress,omitempty"`
	TxHash          string `json:"txHash,omitempty"`
}

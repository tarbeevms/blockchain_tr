package models

type VoteRequest struct {
	Voter       string `json:"voter"`
	CandidateID uint64 `json:"candidateId"`
}

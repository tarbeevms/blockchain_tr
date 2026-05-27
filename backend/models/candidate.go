package models

type Candidate struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	VoteCount uint64 `json:"voteCount"`
}

type CandidateRequest struct {
	Name string `json:"name"`
}

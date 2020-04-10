package main

type TwitterActivity struct {
	Action      string          `json:"action"`
	MaxPosition uint64          `json:"max_position"`
	Sources     []TwitterUser   `json:"sources,omitempty"`
	Target      []TwitterStatus `json:"targets"`
}

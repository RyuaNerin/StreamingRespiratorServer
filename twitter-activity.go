package main

type TwitterActivity struct {
	Action      string          `json:"action"`
	MaxPosition string          `json:"max_position"`
	Sources     []TwitterUser   `json:"sources"`
	Target      []TwitterStatus `json:"targets"`
}

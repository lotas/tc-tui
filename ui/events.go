package ui

type UIEvent int64

const (
	Quit UIEvent = iota
	Signin
	ListWorkers
	ListWorkerPools
	ListRoles
	ListScopes
)

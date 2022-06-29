package ui

type UIEvent int64

const (
	Quit UIEvent = iota
	Signin
	ListWorkers
	ListWorkerPools
	ShowWorkerPool
	ListRoles
	ShowRole
	ListScopes
)

type EventPayload struct {
	Index int
	Title string
}

type EventCallback func(UIEvent, EventPayload)

type UIPage string

const (
	Info UIPage = "info"
	Menu UIPage = "menu"
)

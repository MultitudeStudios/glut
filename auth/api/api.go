package api

import (
	"glut/auth"
	"glut/common/flux"
)

type API struct {
	service *auth.Service
}

func Handler(s *flux.Server, service *auth.Service) {
	a := &API{service}

	// Sessions API
	flux.New(s, "auth.sessions.query", a.QuerySessions, &flux.Options{})
	flux.New(s, "auth.sessions.create", a.CreateSession, &flux.Options{})
	flux.New(s, "auth.sessions.clear", a.ClearSessions, &flux.Options{})

	// Users API
	flux.New(s, "auth.users.query", a.QueryUsers, &flux.Options{})
	flux.New(s, "auth.users.create", a.CreateUser, &flux.Options{})
}

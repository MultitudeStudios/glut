package api

import (
	"glut/auth"
	"glut/common/flux"
)

func Handler(s *flux.Server, service *auth.Service) {
	// Sessions API
	s.Handle("auth.sessions.query", querySessions(service), &flux.Options{})
	s.Handle("auth.sessions.create", createSession(service), &flux.Options{})
	s.Handle("auth.sessions.clear", clearSessions(service), &flux.Options{})

	// Users API
	s.Handle("auth.users.query", queryUsers(service), &flux.Options{})
	s.Handle("auth.users.create", createUser(service), &flux.Options{})
	s.Handle("auth.users.delete", deleteUsers(service), &flux.Options{})

	// Me API
	s.Handle("auth.me.user", myUser(service), &flux.Options{})
	s.Handle("auth.me.deleteUser", deleteMyUser(service), &flux.Options{})
	s.Handle("auth.me.sessions", mySessions(service), &flux.Options{})
	s.Handle("auth.me.logout", logout(service), &flux.Options{})
	s.Handle("auth.me.renewSession", renewSession(service), &flux.Options{})

	// Admin API
	s.Handle("auth.admin.changePassword", changePassword(service), &flux.Options{})
	s.Handle("auth.admin.changeEmail", changeEmail(service), &flux.Options{})
	s.Handle("auth.admin.verifyUser", verifyUser(service), &flux.Options{})
	s.Handle("auth.admin.resetPassword", resetPassword(service), &flux.Options{})

	// Security API
	s.Handle("auth.security.bans", queryBans(service), &flux.Options{})
	s.Handle("auth.security.banUser", banUser(service), &flux.Options{})
	s.Handle("auth.security.unbanUser", unbanUser(service), &flux.Options{})
}

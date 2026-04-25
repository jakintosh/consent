package api

import (
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
)

type User struct {
	Subject string   `json:"subject"`
	Handle  string   `json:"username"`
	Roles   []string `json:"roles"`
}

type CreateUserRequest struct {
	Handle   string   `json:"username"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

type UpdateUserRequest struct {
	Handle *string   `json:"username,omitempty"`
	Roles  *[]string `json:"roles,omitempty"`
}

func userFromDomain(user service.User) User {
	return User{
		Subject: user.Subject,
		Handle:  user.Handle,
		Roles:   append([]string(nil), user.Roles...),
	}
}

func usersFromDomain(users []service.User) []User {
	apiUsers := make([]User, 0, len(users))
	for _, user := range users {
		apiUsers = append(apiUsers, userFromDomain(user))
	}
	return apiUsers
}

func (a *API) buildUsersRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET    /", a.handleListUsers)
	mux.HandleFunc("POST   /", a.handleCreateUser)

	mux.HandleFunc("GET    /{subject}", a.handleGetUser)
	mux.HandleFunc("PATCH  /{subject}", a.handleUpdateUser)
	mux.HandleFunc("DELETE /{subject}", a.handleDeleteUser)

	return mux
}

func (a *API) handleCreateUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRequest[CreateUserRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	user, err := a.service.CreateUser(req.Handle, req.Password, req.Roles)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, userFromDomain(*user))
}

func (a *API) handleGetUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	subject := r.PathValue("subject")
	if subject == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing user subject")
		return
	}

	user, err := a.service.GetUser(subject)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, userFromDomain(*user))
}

func (a *API) handleListUsers(
	w http.ResponseWriter,
	r *http.Request,
) {
	users, err := a.service.ListUsers()
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, usersFromDomain(users))
}

func (a *API) handleUpdateUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	subject := r.PathValue("subject")
	if subject == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing user subject")
		return
	}

	req, err := decodeRequest[UpdateUserRequest](r)
	if err != nil {
		wire.WriteError(w, http.StatusBadRequest, "Malformed JSON")
		return
	}

	user, err := a.service.UpdateUser(subject, &service.UserUpdate{Handle: req.Handle, Roles: req.Roles})
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, userFromDomain(*user))
}

func (a *API) handleDeleteUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	subject := r.PathValue("subject")
	if subject == "" {
		wire.WriteError(w, http.StatusBadRequest, "Missing user subject")
		return
	}

	err := a.service.DeleteUser(subject)
	if err != nil {
		wire.WriteError(w, httpStatusFromError(err), err.Error())
		return
	}

	wire.WriteData(w, http.StatusOK, nil)
}

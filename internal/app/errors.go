package app

import "net/http"

type appErrorKind int

const (
	errAuthorizeRequestInvalid appErrorKind = iota
	errAuthorizePrepare
	errAuthorizeAutoApprove
	errAuthorizeFormInvalid
	errAuthorizeSubmitInvalid
	errAuthorizeCSRFExpired
	errAuthorizeDeny
	errAuthorizeActionMissing
	errAuthorizeDecision
	errAuthorizeApprove
	errLoginFormInvalid
	errLoginFailed
	errHomeSessionUI
)

type appError struct {
	kind  appErrorKind
	cause error
}

type appErrorSpec struct {
	status     int
	title      string
	message    string
	logMessage string
	loggable   bool
}

var appErrorSpecs = map[appErrorKind]appErrorSpec{
	errAuthorizeRequestInvalid: {
		status:     http.StatusBadRequest,
		title:      "Bad Request",
		message:    "That authorization request is missing required details or uses unsupported values.",
		logMessage: "invalid authorization request",
		loggable:   true,
	},
	errAuthorizePrepare: {
		status:     http.StatusInternalServerError,
		title:      "Server Error",
		message:    "The authorization request could not be prepared right now.",
		logMessage: "failed to prepare authorization decision",
		loggable:   true,
	},
	errAuthorizeAutoApprove: {
		status:     http.StatusInternalServerError,
		title:      "Server Error",
		message:    "Automatic authorization could not be completed right now.",
		logMessage: "failed to issue authorization code",
		loggable:   true,
	},
	errAuthorizeFormInvalid: {
		status:     http.StatusBadRequest,
		title:      "Bad Request",
		message:    "That authorization form could not be processed.",
		logMessage: "failed to parse authorize form",
		loggable:   true,
	},
	errAuthorizeSubmitInvalid: {
		status:     http.StatusBadRequest,
		title:      "Bad Request",
		message:    "That authorization decision is missing required details.",
		logMessage: "invalid authorization submit",
		loggable:   true,
	},
	errAuthorizeCSRFExpired: {
		status:   http.StatusForbidden,
		title:    "Action Expired",
		message:  "This approval form is no longer valid. Reload the page and try again.",
		loggable: false,
	},
	errAuthorizeDeny: {
		status:     http.StatusInternalServerError,
		title:      "Server Error",
		message:    "The authorization denial could not be completed right now.",
		logMessage: "failed to deny authorization",
		loggable:   true,
	},
	errAuthorizeActionMissing: {
		status:   http.StatusBadRequest,
		title:    "Bad Request",
		message:  "Choose whether to approve or deny the request.",
		loggable: false,
	},
	errAuthorizeDecision: {
		status:     http.StatusInternalServerError,
		title:      "Server Error",
		message:    "The authorization request could not be verified right now.",
		logMessage: "failed to compute authorization decision",
		loggable:   true,
	},
	errAuthorizeApprove: {
		status:     http.StatusInternalServerError,
		title:      "Server Error",
		message:    "The authorization approval could not be completed right now.",
		logMessage: "failed to approve authorization",
		loggable:   true,
	},
	errLoginFormInvalid: {
		status:     http.StatusBadRequest,
		title:      "Bad Request",
		message:    "That login request could not be processed.",
		logMessage: "failed to parse login form",
		loggable:   true,
	},
	errLoginFailed: {
		status:     http.StatusInternalServerError,
		title:      "Server Error",
		message:    "Login could not be completed right now. Try again in a moment.",
		logMessage: "failed to complete login",
		loggable:   true,
	},
	errHomeSessionUI: {
		status:     http.StatusInternalServerError,
		title:      "Server Error",
		message:    "The session UI could not be prepared right now.",
		logMessage: "failed to build logout URL",
		loggable:   true,
	},
}

func appErr(kind appErrorKind, err error) *appError {
	return &appError{kind: kind, cause: err}
}

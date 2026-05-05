package service_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestLogin_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// returns redirect URL with auth_code
	redirectURL, err := env.Service.Login("alice", "password123", service.InternalIntegrationName)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if redirectURL == nil {
		t.Fatal("expected redirect URL")
	}
	authCode := redirectURL.Query().Get("auth_code")
	if authCode == "" {
		t.Error("redirect URL missing auth_code parameter")
	}
}

func TestLogin_RedirectURL(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// redirects to the integration's configured callback URL
	redirectURL, err := env.Service.Login("alice", "password123", service.InternalIntegrationName)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if redirectURL.Host != "consent.test" {
		t.Errorf("redirect host = %s, want consent.test", redirectURL.Host)
	}
	if redirectURL.Path != "/auth/callback" {
		t.Errorf("redirect path = %s, want /auth/callback", redirectURL.Path)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// wrong password returns ErrInvalidCredentials
	_, err := env.Service.Login("alice", "wrongpassword", service.InternalIntegrationName)
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// unknown user returns ErrAccountNotFound
	_, err := env.Service.Login("unknown", "password", service.InternalIntegrationName)
	if !errors.Is(err, service.ErrAccountNotFound) {
		t.Errorf("expected ErrAccountNotFound, got %v", err)
	}
}

func TestLogin_UnknownIntegration(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// unknown integration returns ErrIntegrationNotFound
	_, err := env.Service.Login("alice", "password123", "nonexistent-service")
	if !errors.Is(err, service.ErrIntegrationNotFound) {
		t.Errorf("expected ErrIntegrationNotFound, got %v", err)
	}
}

func TestLogin_StoresRefreshToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// grant auth_code and get redirect
	redirectURL, err := env.Service.Login("alice", "password123", service.InternalIntegrationName)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	authCode := redirectURL.Query().Get("auth_code")

	// token is stored and owner can be retrieved
	owner, err := env.DB.GetRefreshTokenOwner(authCode)
	if err != nil {
		t.Fatalf("Token not stored: %v", err)
	}
	user, err := env.DB.GetUserByHandle("alice")
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}
	if owner != user.Subject {
		t.Errorf("token owner = %s, want %s", owner, user.Subject)
	}
}

func TestLogin_AuthCodeIsValidJWT(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// grant auth_code and get redirect
	redirectURL, err := env.Service.Login("alice", "password123", service.InternalIntegrationName)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	authCode := redirectURL.Query().Get("auth_code")

	// auth_code is valid JWT format (3 dot-separated parts)
	parts := strings.Split(authCode, ".")
	if len(parts) != 3 {
		t.Errorf("auth_code not valid JWT format, has %d parts", len(parts))
	}
}

func TestLogin_ReturnTo(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password123")

	returnTo := "/authorize?integration=test-integration&scope=identity"
	redirectURL, err := env.Service.Login(
		"alice",
		"password123",
		service.InternalIntegrationName,
		returnTo,
	)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	query := redirectURL.Query()
	if query.Get("auth_code") == "" {
		t.Fatal("redirect URL missing auth_code parameter")
	}
	if query.Get("return_to") != returnTo {
		t.Fatalf("return_to = %q, want %q", query.Get("return_to"), returnTo)
	}
	if query.Get("state") != "" {
		t.Fatalf("state = %q, want empty", query.Get("state"))
	}
}

func TestReviewAuthorizationRequest_MissingScopes(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	env.CreateTestIntegration(t, "test-integration", "Test Integration", "test-audience", "https://app.test/callback", "https://app.test", "https://app.test/logo.png", nil)
	user := getTestUser(t, env, "alice")
	if err := env.DB.InsertGrants(user.Subject, "test-integration", []string{service.ScopeIdentity}); err != nil {
		t.Fatalf("InsertGrants failed: %v", err)
	}

	review, err := env.Service.ReviewAuthorizationRequest(
		user.Subject,
		"test-integration",
		[]string{service.ScopeProfile, service.ScopeIdentity},
		"state-123",
	)
	if err != nil {
		t.Fatalf("ReviewAuthorizationRequest failed: %v", err)
	}

	if review.Request.Integration.Name != "test-integration" {
		t.Fatalf("integration = %q, want test-integration", review.Request.Integration.Name)
	}
	if review.Request.State != "state-123" {
		t.Fatalf("state = %q, want state-123", review.Request.State)
	}
	assertStringsEqual(t, review.Request.Scopes, []string{service.ScopeIdentity, service.ScopeProfile})
	assertScopeDefinitions(t, review.RequestedScopes, []string{service.ScopeIdentity, service.ScopeProfile})
	assertScopeDefinitions(t, review.GrantedScopes, []string{service.ScopeIdentity})
	assertScopeDefinitions(t, review.MissingScopes, []string{service.ScopeProfile})
	if review.IsAuthorized() {
		t.Fatal("expected review to require authorization")
	}
}

func TestReviewAuthorizationRequest_AuthorizedWhenAllScopesGranted(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	env.CreateTestIntegration(t, "test-integration", "Test Integration", "test-audience", "https://app.test/callback", "https://app.test", "https://app.test/logo.png", nil)
	user := getTestUser(t, env, "alice")
	if err := env.DB.InsertGrants(user.Subject, "test-integration", []string{service.ScopeIdentity}); err != nil {
		t.Fatalf("InsertGrants failed: %v", err)
	}

	review, err := env.Service.ReviewAuthorizationRequest(
		user.Subject,
		"test-integration",
		[]string{service.ScopeIdentity},
		"",
	)
	if err != nil {
		t.Fatalf("ReviewAuthorizationRequest failed: %v", err)
	}

	assertScopeDefinitions(t, review.MissingScopes, nil)
	if !review.IsAuthorized() {
		t.Fatal("expected review to already be authorized")
	}
}

func TestFinalizeAuthorization_RedirectIncludesAuthCodeStateAndPreservesQuery(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	env.CreateTestIntegration(t, "test-integration", "Test Integration", "test-audience", "https://app.test/callback?existing=1", "https://app.test", "https://app.test/logo.png", nil)
	user := getTestUser(t, env, "alice")

	review, err := env.Service.ReviewAuthorizationRequest(
		user.Subject,
		"test-integration",
		[]string{service.ScopeIdentity},
		"state-123",
	)
	if err != nil {
		t.Fatalf("ReviewAuthorizationRequest failed: %v", err)
	}
	redirectURL, err := env.Service.FinalizeAuthorization(user.Subject, review)
	if err != nil {
		t.Fatalf("FinalizeAuthorization failed: %v", err)
	}

	if redirectURL.Host != "app.test" {
		t.Fatalf("redirect host = %q, want app.test", redirectURL.Host)
	}
	if redirectURL.Path != "/callback" {
		t.Fatalf("redirect path = %q, want /callback", redirectURL.Path)
	}
	query := redirectURL.Query()
	if query.Get("existing") != "1" {
		t.Fatalf("existing = %q, want 1", query.Get("existing"))
	}
	if query.Get("auth_code") == "" {
		t.Fatal("redirect URL missing auth_code parameter")
	}
	if query.Get("state") != "state-123" {
		t.Fatalf("state = %q, want state-123", query.Get("state"))
	}
	if query.Get("return_to") != "" {
		t.Fatalf("return_to = %q, want empty", query.Get("return_to"))
	}
}

func TestFinalizeAuthorization_StoresAuthCodeAndGrantsMissingScopes(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	env.CreateTestIntegration(t, "test-integration", "Test Integration", "test-audience", "https://app.test/callback", "https://app.test", "https://app.test/logo.png", nil)
	user := getTestUser(t, env, "alice")

	review, err := env.Service.ReviewAuthorizationRequest(
		user.Subject,
		"test-integration",
		[]string{service.ScopeProfile, service.ScopeIdentity},
		"",
	)
	if err != nil {
		t.Fatalf("ReviewAuthorizationRequest failed: %v", err)
	}
	redirectURL, err := env.Service.FinalizeAuthorization(user.Subject, review)
	if err != nil {
		t.Fatalf("FinalizeAuthorization failed: %v", err)
	}

	authCode := redirectURL.Query().Get("auth_code")
	if authCode == "" {
		t.Fatal("redirect URL missing auth_code parameter")
	}
	owner, err := env.DB.GetRefreshTokenOwner(authCode)
	if err != nil {
		t.Fatalf("GetRefreshTokenOwner failed: %v", err)
	}
	if owner != user.Subject {
		t.Fatalf("auth code owner = %q, want %q", owner, user.Subject)
	}

	grants, err := env.DB.ListGrantedScopeNames(user.Subject, "test-integration")
	if err != nil {
		t.Fatalf("ListGrantedScopeNames failed: %v", err)
	}
	assertStringsEqual(t, grants, []string{service.ScopeIdentity, service.ScopeProfile})
}

func TestFinalizeAuthorization_OmitsEmptyState(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	env.CreateTestIntegration(t, "test-integration", "Test Integration", "test-audience", "https://app.test/callback", "https://app.test", "https://app.test/logo.png", nil)
	user := getTestUser(t, env, "alice")

	review, err := env.Service.ReviewAuthorizationRequest(
		user.Subject,
		"test-integration",
		[]string{service.ScopeIdentity},
		"",
	)
	if err != nil {
		t.Fatalf("ReviewAuthorizationRequest failed: %v", err)
	}
	redirectURL, err := env.Service.FinalizeAuthorization(user.Subject, review)
	if err != nil {
		t.Fatalf("FinalizeAuthorization failed: %v", err)
	}

	query := redirectURL.Query()
	if query.Get("auth_code") == "" {
		t.Fatal("redirect URL missing auth_code parameter")
	}
	if query.Get("state") != "" {
		t.Fatalf("state = %q, want empty", query.Get("state"))
	}
}

func TestFinalizeAuthorization_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	user := getTestUser(t, env, "alice")
	review := &service.AuthorizationReview{
		Request: service.AuthorizationRequest{
			Integration: service.Integration{
				Name:     "test-integration",
				Audience: "test-audience",
				Redirect: "bad-url",
			},
			Scopes: []string{service.ScopeIdentity},
		},
	}

	_, err := env.Service.FinalizeAuthorization(user.Subject, review)
	if !errors.Is(err, service.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", err)
	}
}

func TestDenyAuthorization_RedirectIncludesAccessDeniedStateAndPreservesQuery(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	review := &service.AuthorizationReview{
		Request: service.AuthorizationRequest{
			Integration: service.Integration{
				Redirect: "https://app.test/callback?existing=1",
			},
			State: "state-123",
		},
	}

	redirectURL, err := env.Service.DenyAuthorization(review)
	if err != nil {
		t.Fatalf("DenyAuthorization failed: %v", err)
	}

	query := redirectURL.Query()
	if query.Get("existing") != "1" {
		t.Fatalf("existing = %q, want 1", query.Get("existing"))
	}
	if query.Get("error") != "access_denied" {
		t.Fatalf("error = %q, want access_denied", query.Get("error"))
	}
	if query.Get("state") != "state-123" {
		t.Fatalf("state = %q, want state-123", query.Get("state"))
	}
	if query.Get("auth_code") != "" {
		t.Fatalf("auth_code = %q, want empty", query.Get("auth_code"))
	}
}

func TestDenyAuthorization_OmitsEmptyState(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	review := &service.AuthorizationReview{
		Request: service.AuthorizationRequest{
			Integration: service.Integration{
				Redirect: "https://app.test/callback",
			},
		},
	}

	redirectURL, err := env.Service.DenyAuthorization(review)
	if err != nil {
		t.Fatalf("DenyAuthorization failed: %v", err)
	}

	query := redirectURL.Query()
	if query.Get("error") != "access_denied" {
		t.Fatalf("error = %q, want access_denied", query.Get("error"))
	}
	if query.Get("state") != "" {
		t.Fatalf("state = %q, want empty", query.Get("state"))
	}
}

func TestDenyAuthorization_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	review := &service.AuthorizationReview{
		Request: service.AuthorizationRequest{
			Integration: service.Integration{Redirect: "bad-url"},
		},
	}

	_, err := env.Service.DenyAuthorization(review)
	if !errors.Is(err, service.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", err)
	}
}

func TestRefreshAccessToken_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// refreshing valid token returns new access and refresh tokens
	accessToken, newRefreshToken, err := env.Service.RefreshAccessToken(token.Encoded())
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}
	if accessToken == "" {
		t.Error("expected non-empty access token")
	}
	if newRefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestRefreshAccessToken_InvalidToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// malformed token returns ErrTokenInvalid
	_, _, err := env.Service.RefreshAccessToken("invalid-token")
	if !errors.Is(err, service.ErrTokenInvalid) {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestRefreshAccessToken_TokenNotInStore(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.IssueTestRefreshToken(t, "alice", []string{"test-audience"})

	// valid token not in store returns ErrTokenNotFound
	_, _, err := env.Service.RefreshAccessToken(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestRefreshAccessToken_DeletesOldToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first refresh succeeds
	_, _, err := env.Service.RefreshAccessToken(token.Encoded())
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}

	// old token is deleted and can't be used again
	_, _, err = env.Service.RefreshAccessToken(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("old token should be deleted, got %v", err)
	}
}

func TestRefreshAccessToken_StoresNewToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// refresh returns new token
	_, newRefreshToken, err := env.Service.RefreshAccessToken(token.Encoded())
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}

	// new token is stored in database
	owner, err := env.DB.GetRefreshTokenOwner(newRefreshToken)
	if err != nil {
		t.Fatalf("new token not stored: %v", err)
	}
	user, err := env.DB.GetUserByHandle("alice")
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}
	if owner != user.Subject {
		t.Errorf("new token owner = %s, want %s", owner, user.Subject)
	}
}

func TestRefreshAccessToken_CanBeRefreshedAgain(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first refresh succeeds
	_, newRefreshToken1, err := env.Service.RefreshAccessToken(token.Encoded())
	if err != nil {
		t.Fatalf("First RefreshAccessToken failed: %v", err)
	}

	// new token can be used for another refresh
	_, newRefreshToken2, err := env.Service.RefreshAccessToken(newRefreshToken1)
	if err != nil {
		t.Fatalf("Second RefreshAccessToken failed: %v", err)
	}
	if newRefreshToken2 == "" {
		t.Error("expected non-empty second refresh token")
	}
}

func TestRefreshAccessToken_AccessDeniedWhenRequiredRoleLost(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestRole(t, "admin-app-user", "Admin App User")
	env.CreateTestIntegration(
		t,
		"admin-app",
		"Admin App",
		"admin-audience",
		"https://admin.test/callback",
		"https://admin.test",
		"https://admin.test/logo.png",
		[]string{"admin-app-user"},
	)
	user, err := env.Service.CreateUser("alice", "password", []string{"admin-app-user"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	roles := []string{}
	if _, err := env.Service.UpdateUser(user.Subject, &service.UserUpdate{Roles: &roles}); err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}
	token := env.StoreTestRefreshToken(t, user.Subject, []string{"test.consent.local", "admin-audience"})

	_, _, err = env.Service.RefreshAccessToken(token.Encoded())
	if !errors.Is(err, service.ErrAccessDenied) {
		t.Fatalf("expected ErrAccessDenied, got %v", err)
	}

	_, err = env.DB.GetRefreshTokenOwner(token.Encoded())
	if err == nil {
		t.Fatal("expected denied refresh token to be consumed")
	}
}

func TestRefreshAccessToken_AllowsUnknownAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"external-audience"})

	_, _, err := env.Service.RefreshAccessToken(token.Encoded())
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}
}

func TestRevokeRefreshToken_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// revoking valid token succeeds
	err := env.Service.RevokeRefreshToken(token.Encoded())
	if err != nil {
		t.Fatalf("RevokeRefreshToken failed: %v", err)
	}
}

func TestRevokeRefreshToken_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// revoking non-existent token returns ErrTokenNotFound
	err := env.Service.RevokeRefreshToken("nonexistent-token")
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestRevokeRefreshToken_CantRefreshAfterRevoke(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// revoke the token
	if err := env.Service.RevokeRefreshToken(token.Encoded()); err != nil {
		t.Fatalf("RevokeRefreshToken failed: %v", err)
	}

	// revoked token can't be used for refresh
	_, _, err := env.Service.RefreshAccessToken(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound after revoke, got %v", err)
	}
}

func TestRevokeRefreshToken_DoubleRevoke(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first revoke succeeds
	if err := env.Service.RevokeRefreshToken(token.Encoded()); err != nil {
		t.Fatalf("First RevokeRefreshToken failed: %v", err)
	}

	// second revoke returns ErrTokenNotFound
	err := env.Service.RevokeRefreshToken(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound on second revoke, got %v", err)
	}
}

func getTestUser(
	t *testing.T,
	env *testutil.TestEnv,
	handle string,
) *service.User {
	t.Helper()
	user, err := env.DB.GetUserByHandle(handle)
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}
	return user
}

func assertScopeDefinitions(
	t *testing.T,
	definitions []service.ScopeDefinition,
	want []string,
) {
	t.Helper()
	got := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		got = append(got, definition.Name)
	}
	assertStringsEqual(t, got, want)
}

func assertStringsEqual(
	t *testing.T,
	got []string,
	want []string,
) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestReviewAuthorizationRequest_AccessDenied(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestRole(t, "editor", "Editor")
	env.CreateTestRole(t, "superadmin", "Admin")

	// Create integration requiring "superadmin" role
	err := env.Service.CreateIntegration(
		"admin-app",
		"Admin App",
		"admin-audience",
		"https://admin.test/callback",
		"https://admin.test",
		"https://admin.test/logo.png",
		[]string{"superadmin"},
	)
	if err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}

	// Create user with "editor" role (not "superadmin")
	editorUser, err := env.Service.CreateUser("editor-user", "password", []string{"editor"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Review should fail with ErrAccessDenied
	_, err = env.Service.ReviewAuthorizationRequest(
		editorUser.Subject,
		"admin-app",
		[]string{service.ScopeIdentity},
		"state123",
	)
	if !errors.Is(err, service.ErrAccessDenied) {
		t.Fatalf("expected ErrAccessDenied, got %v", err)
	}
}

func TestReviewAuthorizationRequest_AccessAllowed(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestRole(t, "superadmin", "Admin")

	// Create integration requiring "superadmin" role
	err := env.Service.CreateIntegration(
		"admin-app",
		"Admin App",
		"admin-audience",
		"https://admin.test/callback",
		"https://admin.test",
		"https://admin.test/logo.png",
		[]string{"superadmin"},
	)
	if err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}

	// Create user with "superadmin" role
	adminUser, err := env.Service.CreateUser("admin-user", "password", []string{"superadmin"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Review should succeed
	review, err := env.Service.ReviewAuthorizationRequest(
		adminUser.Subject,
		"admin-app",
		[]string{service.ScopeIdentity},
		"state123",
	)
	if err != nil {
		t.Fatalf("ReviewAuthorizationRequest failed: %v", err)
	}
	if review == nil {
		t.Fatal("expected non-nil review")
	}
	if review.Request.Integration.Name != "admin-app" {
		t.Errorf("Integration.Name = %s, want admin-app", review.Request.Integration.Name)
	}
}

func TestReviewAuthorizationRequest_NoRequiredRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// Integration with no required roles should be accessible to any user
	env.CreateTestIntegration(t, "open-app", "Open App", "open-audience", "https://open.test/callback", "https://open.test", "https://open.test/logo.png", nil)

	// User without any roles should still be able to review
	noRoleUser, err := env.Service.CreateUser("no-role-user", "password", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	review, err := env.Service.ReviewAuthorizationRequest(
		noRoleUser.Subject,
		"open-app",
		[]string{service.ScopeIdentity},
		"state123",
	)
	if err != nil {
		t.Fatalf("ReviewAuthorizationRequest failed: %v", err)
	}
	if review == nil {
		t.Fatal("expected non-nil review")
	}
}

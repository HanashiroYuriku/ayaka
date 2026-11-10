package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
	"be-ayaka/internal/core/service"
	"be-ayaka/internal/testingutils/mocks"
	"be-ayaka/pkg/securetoken"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var any_ = mock.Anything

type AuthServiceSuite struct {
	suite.Suite
	ctx     context.Context
	users   *mocks.UserRepo
	refresh *mocks.RefreshTokenRepo
	tokens  *mocks.UserTokenRepo
	hasher  *mocks.Hasher
	access  *mocks.AccessTokenProvider
	mailer  *mocks.AuthMailer
	tx      *mocks.FakeTx
	cfg     service.AuthConfig
	svc     service.AuthService
}

func TestAuthServiceSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceSuite))
}

func (s *AuthServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.users = new(mocks.UserRepo)
	s.refresh = new(mocks.RefreshTokenRepo)
	s.tokens = new(mocks.UserTokenRepo)
	s.hasher = new(mocks.Hasher)
	s.access = new(mocks.AccessTokenProvider)
	s.mailer = new(mocks.AuthMailer)
	s.tx = new(mocks.FakeTx)
	s.cfg = service.AuthConfig{
		RefreshTTL:               168 * time.Hour,
		RequireEmailVerification: true,
		MaxFailedLogins:          5,
		LockoutDuration:          15 * time.Minute,
		PasswordResetTTL:         30 * time.Minute,
		EmailVerificationTTL:     24 * time.Hour,
		EmailCooldown:            time.Minute,
		ResetPasswordURL:         "http://app.test/reset-password",
		VerifyEmailURL:           "http://app.test/verify-email",
	}

	// NewAuthService pre-computes a dummy hash for timing-safe logins.
	s.hasher.On("Hash", "ayaka-dummy-password").Return("DUMMY-HASH", nil).Once()
	s.newService()
}

func (s *AuthServiceSuite) newService() {
	s.svc = service.NewAuthService(service.AuthDeps{
		Users: s.users, RefreshTokens: s.refresh, UserTokens: s.tokens,
		Hasher: s.hasher, AccessTokens: s.access, Mailer: s.mailer, Tx: s.tx,
	}, s.cfg)
}

func (s *AuthServiceSuite) TearDownTest() {
	for _, m := range []interface{ AssertExpectations(mock.TestingT) bool }{
		s.users, s.refresh, s.tokens, s.hasher, s.access, s.mailer,
	} {
		m.AssertExpectations(s.T())
	}
}

// ---------------------------------------------------------------- fixtures

func verifiedUser() *entity.User {
	now := time.Now()
	u := &entity.User{Username: "riku", Email: "riku@mail.com", Password: "HASHED", Role: entity.RoleUser, EmailVerifiedAt: &now}
	u.ID = "USR-1"
	return u
}

// expectSession expects a new refresh token + access token for user in a session.
func (s *AuthServiceSuite) expectSession(user *entity.User, sessionID any) {
	s.refresh.On("Create", any_, mock.MatchedBy(func(rt *entity.RefreshToken) bool {
		return rt.UserID == user.ID && len(rt.TokenHash) == 64 && rt.ExpiresAt.After(time.Now().Add(167*time.Hour))
	})).Return(nil).Once()
	s.access.On("Issue", user.ID, user.Role, sessionID).Return("ACCESS", time.Now().Add(15*time.Minute), nil).Once()
}

func linkWithToken(prefix string) any {
	return mock.MatchedBy(func(link string) bool {
		return strings.HasPrefix(link, prefix+"?token=") && len(link) > len(prefix)+40
	})
}

// ============================================================ Register

func (s *AuthServiceSuite) TestRegister_CreatesUserAndSendsVerification() {
	s.hasher.On("Hash", "P4$$word").Return("HASHED", nil).Once()
	s.users.On("Create", any_, mock.MatchedBy(func(u *entity.User) bool {
		return u.Username == "riku" && u.Email == "riku@mail.com" && u.Password == "HASHED" &&
			u.Role == entity.RoleUser && u.EmailVerifiedAt == nil && u.ID != ""
	})).Return(nil).Once()
	s.tokens.On("Create", any_, mock.MatchedBy(func(t *entity.UserToken) bool {
		return t.Purpose == entity.PurposeEmailVerification && len(t.TokenHash) == 64
	})).Return(nil).Once()
	s.mailer.On("SendEmailVerification", any_, "riku@mail.com", "riku", linkWithToken("http://app.test/verify-email"), 24*time.Hour).
		Return(nil).Once()

	profile, err := s.svc.Register(s.ctx, model.RegisterInput{Username: " riku ", Email: " Riku@Mail.com ", Password: "P4$$word"})

	s.Require().NoError(err)
	s.False(profile.EmailVerified)
	s.Equal(1, s.tx.Count)
}

func (s *AuthServiceSuite) TestRegister_Conflict_SendsNoEmail() {
	s.hasher.On("Hash", any_).Return("HASHED", nil).Once()
	s.users.On("Create", any_, any_).Return(apperror.Conflict("username or email already exists")).Once()

	_, err := s.svc.Register(s.ctx, model.RegisterInput{Password: "x"})

	s.ErrorIs(err, apperror.ErrConflict)
	s.mailer.AssertNotCalled(s.T(), "SendEmailVerification", any_, any_, any_, any_, any_)
}

func (s *AuthServiceSuite) TestRegister_HashFailure_IsInternal() {
	s.hasher.On("Hash", "x").Return("", errors.New("boom")).Once()

	_, err := s.svc.Register(s.ctx, model.RegisterInput{Password: "x"})

	s.ErrorIs(err, apperror.ErrInternal)
}

func (s *AuthServiceSuite) TestCreateAdmin_IsVerifiedAdmin() {
	s.hasher.On("Hash", "P4$$word").Return("HASHED", nil).Once()
	s.users.On("Create", any_, mock.MatchedBy(func(u *entity.User) bool {
		return u.Role == entity.RoleAdmin && u.EmailVerifiedAt != nil
	})).Return(nil).Once()

	profile, err := s.svc.CreateAdmin(s.ctx, model.RegisterInput{Username: "admin", Email: "admin@mail.com", Password: "P4$$word"})

	s.Require().NoError(err)
	s.True(profile.EmailVerified)
}

// ============================================================ Login

func (s *AuthServiceSuite) TestLogin_Success_StartsSession() {
	user := verifiedUser()
	s.users.On("FindByEmail", any_, "riku@mail.com").Return(user, nil).Once()
	s.hasher.On("Compare", "HASHED", "P4$$word").Return(nil).Once()
	s.expectSession(user, mock.MatchedBy(func(sid string) bool { return strings.HasPrefix(sid, "SES-") }))

	tokens, err := s.svc.Login(s.ctx, model.LoginInput{Email: "RIKU@mail.com", Password: "P4$$word"})

	s.Require().NoError(err)
	s.Equal("ACCESS", tokens.AccessToken)
	s.Len(tokens.RefreshToken, 43, "opaque 256-bit token")
	s.Equal("Bearer", tokens.TokenType)
}

func (s *AuthServiceSuite) TestLogin_UnknownEmail_ComparesDummyHash() {
	s.users.On("FindByEmail", any_, "ghost@mail.com").Return(nil, apperror.NotFound("user not found")).Once()
	s.hasher.On("Compare", "DUMMY-HASH", "whatever").Return(errors.New("mismatch")).Once()

	_, err := s.svc.Login(s.ctx, model.LoginInput{Email: "ghost@mail.com", Password: "whatever"})

	s.ErrorIs(err, service.ErrInvalidCredentials)
}

func (s *AuthServiceSuite) TestLogin_WrongPassword_RecordsFailure() {
	user := verifiedUser()
	s.users.On("FindByEmail", any_, "riku@mail.com").Return(user, nil).Once()
	s.hasher.On("Compare", "HASHED", "wrong").Return(errors.New("mismatch")).Once()
	s.users.On("RegisterFailedLogin", any_, "USR-1", 5, mock.MatchedBy(func(t time.Time) bool {
		return t.After(time.Now().Add(14 * time.Minute))
	})).Return(nil).Once()

	_, err := s.svc.Login(s.ctx, model.LoginInput{Email: "riku@mail.com", Password: "wrong"})

	s.ErrorIs(err, service.ErrInvalidCredentials)
	s.access.AssertNotCalled(s.T(), "Issue", any_, any_, any_)
}

func (s *AuthServiceSuite) TestLogin_LockedAccount_DoesNotCheckPassword() {
	user := verifiedUser()
	until := time.Now().Add(10 * time.Minute)
	user.LockedUntil = &until
	s.users.On("FindByEmail", any_, "riku@mail.com").Return(user, nil).Once()

	_, err := s.svc.Login(s.ctx, model.LoginInput{Email: "riku@mail.com", Password: "P4$$word"})

	s.ErrorIs(err, service.ErrAccountLocked)
	s.hasher.AssertNotCalled(s.T(), "Compare", any_, any_)
}

func (s *AuthServiceSuite) TestLogin_ExpiredLock_ResetsCounter() {
	user := verifiedUser()
	past := time.Now().Add(-time.Minute)
	user.LockedUntil, user.FailedLoginAttempts = &past, 2
	s.users.On("FindByEmail", any_, "riku@mail.com").Return(user, nil).Once()
	s.hasher.On("Compare", "HASHED", "P4$$word").Return(nil).Once()
	s.users.On("ResetFailedLogins", any_, "USR-1").Return(nil).Once()
	s.expectSession(user, any_)

	_, err := s.svc.Login(s.ctx, model.LoginInput{Email: "riku@mail.com", Password: "P4$$word"})

	s.NoError(err)
}

func (s *AuthServiceSuite) TestLogin_UnverifiedEmail_Forbidden() {
	user := verifiedUser()
	user.EmailVerifiedAt = nil
	s.users.On("FindByEmail", any_, "riku@mail.com").Return(user, nil).Once()
	s.hasher.On("Compare", "HASHED", "P4$$word").Return(nil).Once()

	_, err := s.svc.Login(s.ctx, model.LoginInput{Email: "riku@mail.com", Password: "P4$$word"})

	s.ErrorIs(err, service.ErrEmailNotVerified)
}

func (s *AuthServiceSuite) TestLogin_UnverifiedEmail_AllowedWhenNotRequired() {
	s.cfg.RequireEmailVerification = false
	s.hasher.On("Hash", "ayaka-dummy-password").Return("DUMMY-HASH", nil).Once()
	s.newService()

	user := verifiedUser()
	user.EmailVerifiedAt = nil
	s.users.On("FindByEmail", any_, "riku@mail.com").Return(user, nil).Once()
	s.hasher.On("Compare", "HASHED", "P4$$word").Return(nil).Once()
	s.expectSession(user, any_)

	_, err := s.svc.Login(s.ctx, model.LoginInput{Email: "riku@mail.com", Password: "P4$$word"})

	s.NoError(err)
}

// ============================================================ Refresh / logout

func (s *AuthServiceSuite) currentRefresh() *entity.RefreshToken {
	return &entity.RefreshToken{
		ID: "RT-1", UserID: "USR-1", FamilyID: "SES-1",
		TokenHash: securetoken.Hash("plain"), ExpiresAt: time.Now().Add(time.Hour),
	}
}

func (s *AuthServiceSuite) TestRefresh_RotatesTokenInSameSession() {
	user := verifiedUser()
	s.refresh.On("FindByHash", any_, securetoken.Hash("plain"), true).Return(s.currentRefresh(), nil).Once()
	s.users.On("FindByID", any_, "USR-1").Return(user, nil).Once()
	s.expectSession(user, "SES-1")
	s.refresh.On("Revoke", any_, "RT-1", mock.MatchedBy(func(id *string) bool {
		return id != nil && strings.HasPrefix(*id, "RT-")
	}), any_).Return(nil).Once()

	tokens, err := s.svc.Refresh(s.ctx, "plain", model.ClientInfo{IP: "127.0.0.1"})

	s.Require().NoError(err)
	s.NotEqual("plain", tokens.RefreshToken)
}

func (s *AuthServiceSuite) TestRefresh_UnknownToken() {
	s.refresh.On("FindByHash", any_, any_, true).Return(nil, apperror.NotFound("token not found")).Once()

	_, err := s.svc.Refresh(s.ctx, "nope", model.ClientInfo{})

	s.ErrorIs(err, service.ErrInvalidToken)
}

func (s *AuthServiceSuite) TestRefresh_ReuseRevokesWholeSession() {
	rt := s.currentRefresh()
	revoked, next := time.Now().Add(-time.Minute), "RT-2"
	rt.RevokedAt, rt.ReplacedBy = &revoked, &next // already rotated
	s.refresh.On("FindByHash", any_, any_, true).Return(rt, nil).Once()
	s.refresh.On("RevokeFamily", any_, "SES-1", any_).Return(nil).Once()

	_, err := s.svc.Refresh(s.ctx, "plain", model.ClientInfo{})

	s.ErrorIs(err, service.ErrInvalidToken)
	s.access.AssertNotCalled(s.T(), "Issue", any_, any_, any_)
}

func (s *AuthServiceSuite) TestRefresh_LoggedOutToken_IsNotTreatedAsReuse() {
	rt := s.currentRefresh()
	revoked := time.Now().Add(-time.Minute)
	rt.RevokedAt = &revoked // revoked by logout: ReplacedBy is nil
	s.refresh.On("FindByHash", any_, any_, true).Return(rt, nil).Once()

	_, err := s.svc.Refresh(s.ctx, "plain", model.ClientInfo{})

	s.ErrorIs(err, service.ErrInvalidToken)
	s.refresh.AssertNotCalled(s.T(), "RevokeFamily", any_, any_, any_)
}

func (s *AuthServiceSuite) TestRefresh_Expired() {
	rt := s.currentRefresh()
	rt.ExpiresAt = time.Now().Add(-time.Second)
	s.refresh.On("FindByHash", any_, any_, true).Return(rt, nil).Once()

	_, err := s.svc.Refresh(s.ctx, "plain", model.ClientInfo{})

	s.ErrorIs(err, service.ErrInvalidToken)
}

func (s *AuthServiceSuite) TestLogout_RevokesSession_AndIsIdempotent() {
	s.refresh.On("FindByHash", any_, securetoken.Hash("plain"), false).Return(s.currentRefresh(), nil).Once()
	s.refresh.On("RevokeFamily", any_, "SES-1", any_).Return(nil).Once()
	s.NoError(s.svc.Logout(s.ctx, "plain"))

	s.refresh.On("FindByHash", any_, securetoken.Hash("unknown"), false).Return(nil, apperror.NotFound("x")).Once()
	s.NoError(s.svc.Logout(s.ctx, "unknown"))
}

func (s *AuthServiceSuite) TestLogoutAll() {
	s.refresh.On("RevokeAllForUser", any_, "USR-1", any_).Return(nil).Once()

	s.NoError(s.svc.LogoutAll(s.ctx, "USR-1"))
}

// ============================================================ Change password

func (s *AuthServiceSuite) TestChangePassword_Success_RevokesSessions() {
	s.users.On("FindByID", any_, "USR-1").Return(verifiedUser(), nil).Once()
	s.hasher.On("Compare", "HASHED", "Old$pass1").Return(nil).Once()
	s.hasher.On("Hash", "New$pass1").Return("NEW-HASH", nil).Once()
	s.users.On("UpdatePassword", any_, "USR-1", "NEW-HASH").Return(nil).Once()
	s.refresh.On("RevokeAllForUser", any_, "USR-1", any_).Return(nil).Once()

	err := s.svc.ChangePassword(s.ctx, model.ChangePasswordInput{UserID: "USR-1", CurrentPassword: "Old$pass1", NewPassword: "New$pass1"})

	s.NoError(err)
}

func (s *AuthServiceSuite) TestChangePassword_WrongCurrent() {
	s.users.On("FindByID", any_, "USR-1").Return(verifiedUser(), nil).Once()
	s.hasher.On("Compare", "HASHED", "bad").Return(errors.New("mismatch")).Once()

	err := s.svc.ChangePassword(s.ctx, model.ChangePasswordInput{UserID: "USR-1", CurrentPassword: "bad", NewPassword: "New$pass1"})

	s.ErrorIs(err, service.ErrWrongCurrentPassword)
	s.Contains(apperror.As(err).Fields, "currentPassword")
}

func (s *AuthServiceSuite) TestChangePassword_SamePassword() {
	s.users.On("FindByID", any_, "USR-1").Return(verifiedUser(), nil).Once()
	s.hasher.On("Compare", "HASHED", "Same$pass1").Return(nil).Once()

	err := s.svc.ChangePassword(s.ctx, model.ChangePasswordInput{UserID: "USR-1", CurrentPassword: "Same$pass1", NewPassword: "Same$pass1"})

	s.ErrorIs(err, service.ErrSamePassword)
}

// ============================================================ Forgot / reset password

func (s *AuthServiceSuite) TestRequestPasswordReset_UnknownEmail_SilentlySucceeds() {
	s.users.On("FindByEmail", any_, "ghost@mail.com").Return(nil, apperror.NotFound("x")).Once()

	s.NoError(s.svc.RequestPasswordReset(s.ctx, "ghost@mail.com"))
	s.mailer.AssertNotCalled(s.T(), "SendPasswordReset", any_, any_, any_, any_, any_)
}

func (s *AuthServiceSuite) TestRequestPasswordReset_SendsLink() {
	s.users.On("FindByEmail", any_, "riku@mail.com").Return(verifiedUser(), nil).Once()
	s.tokens.On("LastCreatedAt", any_, "USR-1", entity.PurposePasswordReset).Return(time.Time{}, false, nil).Once()
	s.tokens.On("InvalidateAll", any_, "USR-1", entity.PurposePasswordReset, any_).Return(nil).Once()
	s.tokens.On("Create", any_, mock.MatchedBy(func(t *entity.UserToken) bool {
		return t.Purpose == entity.PurposePasswordReset && t.ExpiresAt.Before(time.Now().Add(31*time.Minute))
	})).Return(nil).Once()
	s.mailer.On("SendPasswordReset", any_, "riku@mail.com", "riku", linkWithToken("http://app.test/reset-password"), 30*time.Minute).
		Return(nil).Once()

	s.NoError(s.svc.RequestPasswordReset(s.ctx, " RIKU@mail.com "))
}

func (s *AuthServiceSuite) TestRequestPasswordReset_CooldownSkipsEmail() {
	s.users.On("FindByEmail", any_, "riku@mail.com").Return(verifiedUser(), nil).Once()
	s.tokens.On("LastCreatedAt", any_, "USR-1", entity.PurposePasswordReset).Return(time.Now().Add(-10*time.Second), true, nil).Once()

	s.NoError(s.svc.RequestPasswordReset(s.ctx, "riku@mail.com"))
	s.tokens.AssertNotCalled(s.T(), "Create", any_, any_)
}

func (s *AuthServiceSuite) usableToken(purpose entity.TokenPurpose) *entity.UserToken {
	return &entity.UserToken{ID: "UTK-1", UserID: "USR-1", Purpose: purpose, ExpiresAt: time.Now().Add(time.Minute)}
}

func (s *AuthServiceSuite) TestResetPassword_Success() {
	user := verifiedUser()
	user.EmailVerifiedAt = nil
	s.tokens.On("FindByHash", any_, entity.PurposePasswordReset, securetoken.Hash("tok")).
		Return(s.usableToken(entity.PurposePasswordReset), nil).Once()
	s.tokens.On("MarkUsed", any_, "UTK-1", any_).Return(nil).Once()
	s.hasher.On("Hash", "New$pass1").Return("NEW-HASH", nil).Once()
	s.users.On("UpdatePassword", any_, "USR-1", "NEW-HASH").Return(nil).Once()
	s.refresh.On("RevokeAllForUser", any_, "USR-1", any_).Return(nil).Once()
	s.tokens.On("InvalidateAll", any_, "USR-1", entity.PurposePasswordReset, any_).Return(nil).Once()
	s.users.On("FindByID", any_, "USR-1").Return(user, nil).Once()
	s.users.On("MarkEmailVerified", any_, "USR-1", any_).Return(nil).Once()

	s.NoError(s.svc.ResetPassword(s.ctx, model.ResetPasswordInput{Token: "tok", NewPassword: "New$pass1"}))
}

func (s *AuthServiceSuite) TestResetPassword_InvalidExpiredOrUsedToken() {
	expired := s.usableToken(entity.PurposePasswordReset)
	expired.ExpiresAt = time.Now().Add(-time.Second)

	s.tokens.On("FindByHash", any_, entity.PurposePasswordReset, securetoken.Hash("unknown")).Return(nil, apperror.NotFound("x")).Once()
	s.tokens.On("FindByHash", any_, entity.PurposePasswordReset, securetoken.Hash("expired")).Return(expired, nil).Once()
	s.tokens.On("FindByHash", any_, entity.PurposePasswordReset, securetoken.Hash("race")).Return(s.usableToken(entity.PurposePasswordReset), nil).Once()
	s.tokens.On("MarkUsed", any_, "UTK-1", any_).Return(apperror.NotFound("already used")).Once()

	for _, tok := range []string{"unknown", "expired", "race"} {
		err := s.svc.ResetPassword(s.ctx, model.ResetPasswordInput{Token: tok, NewPassword: "New$pass1"})
		s.ErrorIs(err, service.ErrInvalidResetToken, tok)
	}
	s.users.AssertNotCalled(s.T(), "UpdatePassword", any_, any_, any_)
}

// ============================================================ Email verification

func (s *AuthServiceSuite) TestVerifyEmail_Success() {
	s.tokens.On("FindByHash", any_, entity.PurposeEmailVerification, securetoken.Hash("tok")).
		Return(s.usableToken(entity.PurposeEmailVerification), nil).Once()
	s.tokens.On("MarkUsed", any_, "UTK-1", any_).Return(nil).Once()
	s.tokens.On("InvalidateAll", any_, "USR-1", entity.PurposeEmailVerification, any_).Return(nil).Once()
	s.users.On("MarkEmailVerified", any_, "USR-1", any_).Return(nil).Once()

	s.NoError(s.svc.VerifyEmail(s.ctx, "tok"))
}

func (s *AuthServiceSuite) TestVerifyEmail_InvalidToken() {
	s.tokens.On("FindByHash", any_, entity.PurposeEmailVerification, any_).Return(nil, apperror.NotFound("x")).Once()

	s.ErrorIs(s.svc.VerifyEmail(s.ctx, "bad"), service.ErrInvalidVerificationToken)
}

func (s *AuthServiceSuite) TestResendVerification_AlreadyVerified_NoEmail() {
	s.users.On("FindByEmail", any_, "riku@mail.com").Return(verifiedUser(), nil).Once()

	s.NoError(s.svc.ResendVerification(s.ctx, "riku@mail.com"))
	s.mailer.AssertNotCalled(s.T(), "SendEmailVerification", any_, any_, any_, any_, any_)
}

func (s *AuthServiceSuite) TestResendVerification_SendsNewLink() {
	user := verifiedUser()
	user.EmailVerifiedAt = nil
	s.users.On("FindByEmail", any_, "riku@mail.com").Return(user, nil).Once()
	s.tokens.On("LastCreatedAt", any_, "USR-1", entity.PurposeEmailVerification).Return(time.Now().Add(-time.Hour), true, nil).Once()
	s.tokens.On("InvalidateAll", any_, "USR-1", entity.PurposeEmailVerification, any_).Return(nil).Once()
	s.tokens.On("Create", any_, any_).Return(nil).Once()
	s.mailer.On("SendEmailVerification", any_, "riku@mail.com", "riku", linkWithToken("http://app.test/verify-email"), 24*time.Hour).
		Return(nil).Once()

	s.NoError(s.svc.ResendVerification(s.ctx, "riku@mail.com"))
}

package repository_test

import (
	"context"
	"testing"
	"time"

	"be-ayaka/internal/adapter/repository"
	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"

	"github.com/stretchr/testify/suite"
)

type TokenRepoSuite struct {
	BaseRepoSuite
	ctx     context.Context
	refresh *repository.RefreshTokenRepo
	tokens  *repository.UserTokenRepo
}

func TestTokenRepoSuite(t *testing.T) {
	suite.Run(t, &TokenRepoSuite{
		BaseRepoSuite: NewBaseRepoSuite(&entity.User{}, &entity.RefreshToken{}, &entity.UserToken{}),
	})
}

func (s *TokenRepoSuite) SetupTest() {
	s.BaseRepoSuite.SetupTest()
	s.ctx = context.Background()
	s.refresh = repository.NewRefreshTokenRepo(s.DB)
	s.tokens = repository.NewUserTokenRepo(s.DB)
	s.Require().NoError(s.DB.Create(newUser("U-1", "riku")).Error)
	s.Require().NoError(s.DB.Create(newUser("U-2", "yuri")).Error)
}

func refreshToken(id, user, family, hash string) *entity.RefreshToken {
	return &entity.RefreshToken{
		ID: id, UserID: user, FamilyID: family, TokenHash: hash,
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}
}

// ============================================================ refresh tokens

func (s *TokenRepoSuite) TestRefresh_CreateFindRevoke() {
	s.Require().NoError(s.refresh.Create(s.ctx, refreshToken("RT-1", "U-1", "F-1", "hash-1")))

	found, err := s.refresh.FindByHash(s.ctx, "hash-1", true) // FOR UPDATE is a no-op on SQLite
	s.Require().NoError(err)
	s.Equal("RT-1", found.ID)

	next := "RT-2"
	s.Require().NoError(s.refresh.Revoke(s.ctx, "RT-1", &next, time.Now()))
	found, _ = s.refresh.FindByHash(s.ctx, "hash-1", false)
	s.True(found.IsRevoked())
	s.Equal("RT-2", *found.ReplacedBy)

	_, err = s.refresh.FindByHash(s.ctx, "unknown", false)
	s.ErrorIs(err, apperror.ErrNotFound)
}

func (s *TokenRepoSuite) TestRefresh_RevokeFamilyAndAllForUser() {
	for _, t := range []*entity.RefreshToken{
		refreshToken("RT-1", "U-1", "F-1", "h1"),
		refreshToken("RT-2", "U-1", "F-1", "h2"),
		refreshToken("RT-3", "U-1", "F-2", "h3"),
		refreshToken("RT-4", "U-2", "F-3", "h4"),
	} {
		s.Require().NoError(s.refresh.Create(s.ctx, t))
	}

	s.Require().NoError(s.refresh.RevokeFamily(s.ctx, "F-1", time.Now()))
	s.revoked(map[string]bool{"h1": true, "h2": true, "h3": false, "h4": false})

	s.Require().NoError(s.refresh.RevokeAllForUser(s.ctx, "U-1", time.Now()))
	s.revoked(map[string]bool{"h3": true, "h4": false})
}

func (s *TokenRepoSuite) revoked(want map[string]bool) {
	for hash, revoked := range want {
		t, err := s.refresh.FindByHash(s.ctx, hash, false)
		s.Require().NoError(err)
		s.Equal(revoked, t.IsRevoked(), hash)
	}
}

func (s *TokenRepoSuite) TestRefresh_DeleteStale() {
	old := refreshToken("RT-1", "U-1", "F-1", "h1")
	old.ExpiresAt = time.Now().Add(-48 * time.Hour)
	s.Require().NoError(s.refresh.Create(s.ctx, old))
	s.Require().NoError(s.refresh.Create(s.ctx, refreshToken("RT-2", "U-1", "F-1", "h2")))

	n, err := s.refresh.DeleteStale(s.ctx, time.Now().Add(-24*time.Hour))

	s.Require().NoError(err)
	s.EqualValues(1, n)
}

// ============================================================ one-time tokens

func userToken(id string, purpose entity.TokenPurpose, hash string, createdAt time.Time) *entity.UserToken {
	return &entity.UserToken{
		ID: id, UserID: "U-1", Purpose: purpose, TokenHash: hash,
		ExpiresAt: createdAt.Add(time.Hour), CreatedAt: createdAt,
	}
}

func (s *TokenRepoSuite) TestUserToken_FindIsScopedByPurpose() {
	s.Require().NoError(s.tokens.Create(s.ctx, userToken("T-1", entity.PurposePasswordReset, "h1", time.Now())))

	_, err := s.tokens.FindByHash(s.ctx, entity.PurposePasswordReset, "h1")
	s.NoError(err)

	_, err = s.tokens.FindByHash(s.ctx, entity.PurposeEmailVerification, "h1")
	s.ErrorIs(err, apperror.ErrNotFound, "a reset token must not verify an email")
}

func (s *TokenRepoSuite) TestUserToken_MarkUsedOnlyOnce() {
	s.Require().NoError(s.tokens.Create(s.ctx, userToken("T-1", entity.PurposePasswordReset, "h1", time.Now())))

	s.NoError(s.tokens.MarkUsed(s.ctx, "T-1", time.Now()))
	s.ErrorIs(s.tokens.MarkUsed(s.ctx, "T-1", time.Now()), apperror.ErrNotFound)
}

func (s *TokenRepoSuite) TestUserToken_InvalidateAllAndLastCreatedAt() {
	_, ok, err := s.tokens.LastCreatedAt(s.ctx, "U-1", entity.PurposePasswordReset)
	s.Require().NoError(err)
	s.False(ok)

	older := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	newer := time.Now().UTC().Truncate(time.Second)
	s.Require().NoError(s.tokens.Create(s.ctx, userToken("T-1", entity.PurposePasswordReset, "h1", older)))
	s.Require().NoError(s.tokens.Create(s.ctx, userToken("T-2", entity.PurposePasswordReset, "h2", newer)))
	s.Require().NoError(s.tokens.Create(s.ctx, userToken("T-3", entity.PurposeEmailVerification, "h3", newer)))

	last, ok, err := s.tokens.LastCreatedAt(s.ctx, "U-1", entity.PurposePasswordReset)
	s.Require().NoError(err)
	s.True(ok)
	s.True(last.Equal(newer))

	s.Require().NoError(s.tokens.InvalidateAll(s.ctx, "U-1", entity.PurposePasswordReset, time.Now()))
	for hash, usable := range map[string]bool{"h1": false, "h2": false} {
		t, _ := s.tokens.FindByHash(s.ctx, entity.PurposePasswordReset, hash)
		s.Equal(usable, t.IsUsable(time.Now()), hash)
	}
	t, _ := s.tokens.FindByHash(s.ctx, entity.PurposeEmailVerification, "h3")
	s.True(t.IsUsable(time.Now()), "other purposes are untouched")
}

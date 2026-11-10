package service

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"time"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/pkg/generateid"
	"be-ayaka/pkg/logger"
	"be-ayaka/pkg/securetoken"
)

var ErrInvalidVerificationToken = apperror.BadRequest("invalid or expired email verification token")

// VerifyEmail confirms the email address using the token from the email.
func (s *authService) VerifyEmail(ctx context.Context, token string) error {
	return s.Tx.WithTx(ctx, func(ctx context.Context) error {
		ut, err := s.redeemUserToken(ctx, entity.PurposeEmailVerification, token)
		if err != nil {
			if errors.Is(err, ErrInvalidToken) {
				return ErrInvalidVerificationToken
			}
			return err
		}
		if err := s.UserTokens.InvalidateAll(ctx, ut.UserID, entity.PurposeEmailVerification, s.now()); err != nil {
			return err
		}
		return s.Users.MarkEmailVerified(ctx, ut.UserID, s.now())
	})
}

// ResendVerification sends a new verification email. Like password reset, it
// never reveals whether the email is registered or already verified.
func (s *authService) ResendVerification(ctx context.Context, email string) error {
	user, err := s.Users.FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil
		}
		return err
	}
	if user.IsEmailVerified() {
		return nil
	}

	allowed, err := s.cooldownPassed(ctx, user.ID, entity.PurposeEmailVerification)
	if err != nil || !allowed {
		return err
	}

	var token string
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.UserTokens.InvalidateAll(ctx, user.ID, entity.PurposeEmailVerification, s.now()); err != nil {
			return err
		}
		token, err = s.issueUserToken(ctx, user.ID, entity.PurposeEmailVerification, s.cfg.EmailVerificationTTL)
		return err
	})
	if err != nil {
		return err
	}

	s.sendVerification(ctx, user, token)
	return nil
}

// sendVerification emails the verification link. Failures are logged, not
// returned: the account exists and the user can ask for a new link.
func (s *authService) sendVerification(ctx context.Context, user *entity.User, token string) {
	err := s.Mailer.SendEmailVerification(ctx, user.Email, user.DisplayNameOrUsername(),
		withToken(s.cfg.VerifyEmailURL, token), s.cfg.EmailVerificationTTL)
	if err != nil {
		logger.FromContext(ctx).Error("send verification email failed",
			slog.String("user_id", user.ID), slog.Any("error", err))
	}
}

// ============================================================ one-time tokens

// issueUserToken stores a new single-use token and returns its plain value.
func (s *authService) issueUserToken(ctx context.Context, userID string, purpose entity.TokenPurpose, ttl time.Duration) (string, error) {
	plain, hash, err := securetoken.New()
	if err != nil {
		return "", apperror.Internal(err)
	}
	now := s.now()
	err = s.UserTokens.Create(ctx, &entity.UserToken{
		ID:        generateid.New("UTK"),
		UserID:    userID,
		Purpose:   purpose,
		TokenHash: hash,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	})
	return plain, err
}

// redeemUserToken validates and consumes a token (exactly once).
func (s *authService) redeemUserToken(ctx context.Context, purpose entity.TokenPurpose, plain string) (*entity.UserToken, error) {
	ut, err := s.UserTokens.FindByHash(ctx, purpose, securetoken.Hash(plain))
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	if !ut.IsUsable(s.now()) {
		return nil, ErrInvalidToken
	}
	if err := s.UserTokens.MarkUsed(ctx, ut.ID, s.now()); err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, ErrInvalidToken // redeemed concurrently
		}
		return nil, err
	}
	return ut, nil
}

// cooldownPassed limits how often emails of one kind are sent to a user
// (prevents mail bombing). Requests inside the cooldown are silently ignored.
func (s *authService) cooldownPassed(ctx context.Context, userID string, purpose entity.TokenPurpose) (bool, error) {
	last, ok, err := s.UserTokens.LastCreatedAt(ctx, userID, purpose)
	if err != nil {
		return false, err
	}
	return !ok || s.now().Sub(last) >= s.cfg.EmailCooldown, nil
}

// withToken appends ?token=... to a frontend URL.
func withToken(base, token string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base + "?token=" + url.QueryEscape(token)
	}
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String()
}

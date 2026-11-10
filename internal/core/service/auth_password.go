package service

import (
	"context"
	"errors"
	"log/slog"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
	"be-ayaka/pkg/logger"
)

var (
	ErrWrongCurrentPassword = apperror.New(apperror.KindValidation, "current password is incorrect").
				WithFields(map[string]string{"currentPassword": "Current password is incorrect"})
	ErrSamePassword = apperror.New(apperror.KindValidation, "new password must be different").
			WithFields(map[string]string{"newPassword": "New password must be different from the current password"})
	ErrInvalidResetToken = apperror.BadRequest("invalid or expired password reset token")
)

// ChangePassword updates the password of a logged-in user and signs out every
// session (including the current one): clients must log in again.
func (s *authService) ChangePassword(ctx context.Context, in model.ChangePasswordInput) error {
	user, err := s.Users.FindByID(ctx, in.UserID)
	if err != nil {
		return err
	}
	if err := s.Hasher.Compare(user.Password, in.CurrentPassword); err != nil {
		return ErrWrongCurrentPassword
	}
	if in.CurrentPassword == in.NewPassword {
		return ErrSamePassword
	}
	return s.setPassword(ctx, user.ID, in.NewPassword)
}

// RequestPasswordReset emails a reset link. It always succeeds from the
// caller's point of view so it cannot be used to discover registered emails.
func (s *authService) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.Users.FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil
		}
		return err
	}

	allowed, err := s.cooldownPassed(ctx, user.ID, entity.PurposePasswordReset)
	if err != nil || !allowed {
		return err
	}

	var token string
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		// Only the newest link works.
		if err := s.UserTokens.InvalidateAll(ctx, user.ID, entity.PurposePasswordReset, s.now()); err != nil {
			return err
		}
		token, err = s.issueUserToken(ctx, user.ID, entity.PurposePasswordReset, s.cfg.PasswordResetTTL)
		return err
	})
	if err != nil {
		return err
	}

	// Delivery failures are logged, not returned: the response must not differ
	// between registered and unknown emails.
	if err := s.Mailer.SendPasswordReset(ctx, user.Email, user.DisplayNameOrUsername(),
		withToken(s.cfg.ResetPasswordURL, token), s.cfg.PasswordResetTTL); err != nil {
		logger.FromContext(ctx).Error("send password reset email failed",
			slog.String("user_id", user.ID), slog.Any("error", err))
	}
	return nil
}

// ResetPassword sets a new password using a reset token, unlocks the account
// and signs out every session.
func (s *authService) ResetPassword(ctx context.Context, in model.ResetPasswordInput) error {
	return s.Tx.WithTx(ctx, func(ctx context.Context) error {
		ut, err := s.redeemUserToken(ctx, entity.PurposePasswordReset, in.Token)
		if err != nil {
			if errors.Is(err, ErrInvalidToken) {
				return ErrInvalidResetToken
			}
			return err
		}

		if err := s.setPassword(ctx, ut.UserID, in.NewPassword); err != nil {
			return err
		}
		if err := s.UserTokens.InvalidateAll(ctx, ut.UserID, entity.PurposePasswordReset, s.now()); err != nil {
			return err
		}

		// Receiving the email proves ownership of the address.
		user, err := s.Users.FindByID(ctx, ut.UserID)
		if err != nil {
			return err
		}
		if !user.IsEmailVerified() {
			return s.Users.MarkEmailVerified(ctx, user.ID, s.now())
		}
		return nil
	})
}

// setPassword hashes and stores the password, then revokes all sessions.
func (s *authService) setPassword(ctx context.Context, userID, password string) error {
	hashed, err := s.Hasher.Hash(password)
	if err != nil {
		return apperror.Internal(err)
	}
	return s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Users.UpdatePassword(ctx, userID, hashed); err != nil {
			return err
		}
		return s.RefreshTokens.RevokeAllForUser(ctx, userID, s.now())
	})
}

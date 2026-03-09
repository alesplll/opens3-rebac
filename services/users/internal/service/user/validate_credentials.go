package user

import (
	"context"
	"errors"

	domainerrors "github.com/alesplll/opens3-rebac/services/users/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/contextx/claimsctx"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/logger"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func (s *userService) ValidateCredentials(ctx context.Context, email, password string) (bool, string) {
	ctx, repoSpan := tracing.StartSpan(ctx, "repo:GetUserCredentials")
	repoSpan.SetAttributes(
		attribute.String("email", email),
	)
	id, storedHash, err := s.repo.GetUserCredentials(ctx, email)
	if err != nil {
		if !errors.Is(err, domainerrors.ErrUserNotFound) {
			repoSpan.SetAttributes(
				attribute.String("result", "failed"),
			)

			logger.Error(claimsctx.InjectUserEmail(ctx, email), "failed to get user credentials", zap.Error(err))
		}
		return false, ""
	}
	repoSpan.SetAttributes(
		attribute.String("result", "success"),
		attribute.String("id", id),
	)
	repoSpan.End()

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		return false, ""
	}

	return true, id
}

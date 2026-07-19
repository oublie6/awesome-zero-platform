package response

import (
	"context"
	"errors"
	"net/http"

	"github.com/oublie6/awesome-zero-platform/server/foundation/apperrors"
	"github.com/oublie6/awesome-zero-platform/server/foundation/requestid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type Envelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
	Data      any    `json:"data"`
}

func InstallHTTPHandlers() {
	httpx.SetOkHandler(func(ctx context.Context, payload any) any {
		if _, ok := payload.(Envelope); ok {
			return payload
		}

		return Success(ctx, payload)
	})

	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, any) {
		appErr := normalizeError(err)
		if appErr.Status() >= http.StatusInternalServerError {
			logx.WithContext(ctx).Errorf("request failed: %v", apperrors.Cause(appErr))
		}

		return appErr.Status(), Error(ctx, appErr)
	})
}

func Success(ctx context.Context, data any) Envelope {
	return Envelope{
		Code:      apperrors.CodeOK,
		Message:   "success",
		RequestID: requestid.FromContext(ctx),
		Data:      data,
	}
}

func Error(ctx context.Context, err error) Envelope {
	normalized := normalizeError(err)
	return Envelope{
		Code:      normalized.Code(),
		Message:   normalized.Message(),
		RequestID: requestid.FromContext(ctx),
		Data:      nil,
	}
}

func WriteError(ctx context.Context, w http.ResponseWriter, err error) {
	normalized := normalizeError(err)
	if normalized.Status() >= http.StatusInternalServerError {
		logx.WithContext(ctx).Errorf("request failed: %v", apperrors.Cause(normalized))
	}

	httpx.WriteJsonCtx(ctx, w, normalized.Status(), Error(ctx, normalized))
}

func WriteJSON(ctx context.Context, w http.ResponseWriter, status int, payload any) {
	httpx.WriteJsonCtx(ctx, w, status, payload)
}

func normalizeError(err error) *apperrors.Error {
	if err == nil {
		return apperrors.Internal(errors.New("nil error"))
	}

	if appErr, ok := apperrors.As(err); ok {
		return appErr
	}

	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return apperrors.RequestTooLarge().WithCause(err)
	}

	return apperrors.Internal(err)
}

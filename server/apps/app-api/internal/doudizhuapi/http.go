package doudizhuapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/securityhttp"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/playing"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore/infrastructure/mysqlarchive"
	"github.com/oublie6/awesome-zero-platform/server/foundation/apperrors"
	platformresponse "github.com/oublie6/awesome-zero-platform/server/foundation/response"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func Register(server *rest.Server, authService *authn.Service, dispatcher *Dispatcher) {
	if server == nil || dispatcher == nil {
		return
	}
	wrap := func(next http.HandlerFunc) http.HandlerFunc {
		return securityhttp.RequireAuthentication(authService, next)
	}
	server.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/v1/doudizhu/commands", Handler: wrap(commandHandler(dispatcher))},
		{Method: http.MethodGet, Path: "/v1/doudizhu/hands/:handId/public", Handler: wrap(publicViewHandler(dispatcher))},
		{Method: http.MethodGet, Path: "/v1/doudizhu/hands/:handId/private", Handler: wrap(privateViewHandler(dispatcher))},
		{Method: http.MethodGet, Path: "/v1/doudizhu/hands/:handId/evidence", Handler: wrap(evidenceHandler(dispatcher))},
	})
}

func commandHandler(dispatcher *Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := authenticatedActor(r)
		if !ok {
			platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("authentication required"))
			return
		}
		var request CommandRequest
		if err := decodeHTTPBody(r, &request); err != nil {
			platformresponse.WriteError(r.Context(), w, apperrors.InvalidParameter("invalid Doudizhu command").WithCause(err))
			return
		}
		result, err := dispatcher.Execute(r.Context(), actor, request)
		if err != nil {
			platformresponse.WriteError(r.Context(), w, mapHTTPError(err))
			return
		}
		platformresponse.WriteJSON(r.Context(), w, http.StatusOK, platformresponse.Success(r.Context(), result))
	}
}

func publicViewHandler(dispatcher *Dispatcher) http.HandlerFunc {
	return viewHandler(dispatcher, false)
}
func privateViewHandler(dispatcher *Dispatcher) http.HandlerFunc {
	return viewHandler(dispatcher, true)
}
func viewHandler(dispatcher *Dispatcher, private bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := authenticatedActor(r)
		if !ok {
			platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("authentication required"))
			return
		}
		handID := domain.HandID(strings.TrimSpace(pathvar.Vars(r)["handId"]))
		if handID == "" {
			platformresponse.WriteError(r.Context(), w, apperrors.InvalidParameter("handId is required"))
			return
		}
		var result ViewResult
		var err error
		if private {
			result, err = dispatcher.PrivateView(r.Context(), actor, handID)
		} else {
			result, err = dispatcher.PublicView(r.Context(), actor, handID)
		}
		if err != nil {
			platformresponse.WriteError(r.Context(), w, mapHTTPError(err))
			return
		}
		platformresponse.WriteJSON(r.Context(), w, http.StatusOK, platformresponse.Success(r.Context(), result))
	}
}

func evidenceHandler(dispatcher *Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := authenticatedActor(r)
		if !ok {
			platformresponse.WriteError(r.Context(), w, apperrors.Unauthorized("authentication required"))
			return
		}
		handID := domain.HandID(strings.TrimSpace(pathvar.Vars(r)["handId"]))
		if handID == "" {
			platformresponse.WriteError(r.Context(), w, apperrors.InvalidParameter("handId is required"))
			return
		}
		result, err := dispatcher.FinalEvidence(r.Context(), actor, handID)
		if err != nil {
			platformresponse.WriteError(r.Context(), w, mapHTTPError(err))
			return
		}
		platformresponse.WriteJSON(r.Context(), w, http.StatusOK, platformresponse.Success(r.Context(), result))
	}
}

func authenticatedActor(r *http.Request) (domain.AccountID, bool) {
	authentication, ok := securityhttp.AuthenticationFromContext(r.Context())
	if !ok || strings.TrimSpace(authentication.Principal.AccountID) == "" {
		return "", false
	}
	return domain.AccountID(authentication.Principal.AccountID), true
}

func decodeHTTPBody(r *http.Request, destination any) error {
	if r == nil || r.Body == nil {
		return ErrInvalidRequest
	}
	payload, err := io.ReadAll(r.Body)
	if err != nil || len(bytes.TrimSpace(payload)) == 0 {
		return ErrInvalidRequest
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return ErrInvalidRequest
	}
	return nil
}

func mapHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, application.ErrInvalidCommand), errors.Is(err, domain.ErrInvalidArgument),
		errors.Is(err, livehand.ErrMalformedCommand), errors.Is(err, livehand.ErrCardNotHeld),
		errors.Is(err, bidding.ErrInvalidScore), errors.Is(err, playing.ErrInvalidPattern), errors.Is(err, playing.ErrInvalidPatternValue):
		return apperrors.InvalidParameter("invalid Doudizhu request").WithCause(err)
	case errors.Is(err, ErrReplayConflict), errors.Is(err, application.ErrOptimisticConflict), errors.Is(err, application.ErrSequenceConflict),
		errors.Is(err, domain.ErrVersionConflict), errors.Is(err, livehand.ErrVersionConflict), errors.Is(err, gamecore.ErrInstanceExists),
		errors.Is(err, gamecore.ErrFinalizationPending), errors.Is(err, bidding.ErrWrongTurn), errors.Is(err, bidding.ErrBidNotHigher),
		errors.Is(err, bidding.ErrBiddingComplete), errors.Is(err, playing.ErrWrongTurn), errors.Is(err, playing.ErrCannotPass),
		errors.Is(err, playing.ErrDoesNotBeat), errors.Is(err, playing.ErrPlayingComplete):
		return apperrors.Conflict("Doudizhu state conflict").WithCause(err)
	case errors.Is(err, domain.ErrNotSeated), errors.Is(err, domain.ErrForbidden), errors.Is(err, application.ErrFinalEvidenceForbidden), errors.Is(err, livehand.ErrViewerNotSeated):
		return apperrors.Forbidden("Doudizhu access denied").WithCause(err)
	case errors.Is(err, application.ErrNotFound), errors.Is(err, gamecore.ErrInstanceNotFound), errors.Is(err, mysqlarchive.ErrArchiveNotFound):
		return apperrors.NotFound("Doudizhu resource not found").WithCause(err)
	default:
		return apperrors.Internal(err)
	}
}

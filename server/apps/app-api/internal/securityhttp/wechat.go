package securityhttp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	"github.com/oublie6/awesome-zero-platform/server/foundation/apperrors"
	platformresponse "github.com/oublie6/awesome-zero-platform/server/foundation/response"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn/adapter/wechatcode"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
	"github.com/zeromicro/go-zero/rest"
)

// Missing platform-owned credentials fail closed; no test identity or simulated
// login can be exchanged for production access tokens.
func registerWeChatCodeLogin(server *rest.Server,ctx *svc.ServiceContext) {
	server.AddRoutes([]rest.Route{{Method:http.MethodPost,Path:"/auth/wechat/code",Handler:wechatCodeHandler(ctx)}})
}
type wechatIdentityVerifier interface {Exchange(context.Context,string)(string,error)}
type wechatAccountBinder interface {ResolveVerified(context.Context,string,string)(identity.Account,error)}
type wechatSessionIssuer interface {
	IssueVerifiedAccount(context.Context,string)(authn.Authentication,authn.TokenPair,error)
}
func wechatCodeHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	var verifier wechatIdentityVerifier
	var binder wechatAccountBinder
	var issuer wechatSessionIssuer
	if ctx!=nil && ctx.Identity!=nil && ctx.MySQL!=nil && ctx.MySQL.DB()!=nil && ctx.Authn!=nil {
		appID:=os.Getenv("APP_WECHAT_APP_ID")
		secret:=os.Getenv("APP_WECHAT_APP_SECRET")
		key,err:=hex.DecodeString(os.Getenv("APP_WECHAT_SUBJECT_HMAC_KEY_HEX"))
		if err==nil && len(key)>=32 {
			verifier,err=wechatcode.New(appID,secret)
			if err==nil {
				binder,err=identity.NewExternalBinding(ctx.MySQL.DB(),ctx.Identity,key)
				if err==nil {issuer=ctx.Authn}
			}
		}
		clear(key)
	}
	return func(w http.ResponseWriter,r *http.Request) {
		if verifier==nil || binder==nil || issuer==nil {
			platformresponse.WriteError(r.Context(),w,apperrors.Internal(errors.New("WeChat identity exchange not configured")))
			return
		}
		var request struct{Code string `json:"code"`}
		dec:=json.NewDecoder(io.LimitReader(r.Body,1024))
		dec.DisallowUnknownFields()
		if err:=dec.Decode(&request);err!=nil || dec.Decode(&struct{}{})!=io.EOF ||
			len(request.Code)<4 || len(request.Code)>128 || strings.TrimSpace(request.Code)!=request.Code {
			platformresponse.WriteError(r.Context(),w,apperrors.InvalidParameter("invalid WeChat code request"))
			return
		}
		openID,err:=verifier.Exchange(r.Context(),request.Code)
		if err!=nil {
			platformresponse.WriteError(r.Context(),w,apperrors.Unauthorized("WeChat login verification failed"))
			return
		}
		account,err:=binder.ResolveVerified(r.Context(),"wechat",openID)
		if err!=nil {
			platformresponse.WriteError(r.Context(),w,apperrors.Internal(errors.New("external identity binding unavailable")))
			return
		}
		authentication,tokens,err:=issuer.IssueVerifiedAccount(r.Context(),account.ID)
		if err!=nil {
			platformresponse.WriteError(r.Context(),w,mapAuthenticationError(err,false))
			return
		}
		writeTokens(r.Context(),w,authentication,tokens)
	}
}

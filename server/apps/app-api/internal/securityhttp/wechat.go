package securityhttp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	"github.com/oublie6/awesome-zero-platform/server/foundation/apperrors"
	platformresponse "github.com/oublie6/awesome-zero-platform/server/foundation/response"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn/adapter/wechatcode"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
	"github.com/zeromicro/go-zero/rest"
)

// A configured WeChat verifier is mandatory. Client-supplied OpenIDs, unionids
// or account IDs are never accepted as proof of login.
func registerWeChatCodeLogin(server *rest.Server,ctx *svc.ServiceContext){
	server.AddRoutes([]rest.Route{{Method:http.MethodPost,Path:"/auth/wechat/code",Handler:wechatCodeHandler(ctx)}})
}
type wechatIdentityVerifier interface {Exchange(context.Context,string)(string,error)}
type wechatAccountBinder interface {ResolveVerified(context.Context,string,string)(identity.Account,error)}
type wechatSessionIssuer interface {
	IssueVerifiedAccount(context.Context,string)(authn.Authentication,authn.TokenPair,error)
}

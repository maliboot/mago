package autoloadx

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/maliboot/mago"
	"github.com/maliboot/mago/config"
)

type Tenant struct {
}

func (m Tenant) UnauthorizedWrapper(ctx context.Context, c *app.RequestContext, code int, message string) {
	mago.NewResponse(c).FailureError(mago.NewError(mago.ErrAuthUnauthorized).WithMsg(message))
}

func (m Tenant) Validate() func(identityVal string) bool {
	return nil
}

func (m Tenant) ResponseWrapper(ctx context.Context, c *app.RequestContext, code int, message map[string]interface{}) {
	mago.NewResponse(c).Success(message)
}

func (m Tenant) IdentityKey() string {
	return "id"
}

func (m Tenant) Login(username, password, ext string) map[string]interface{} {
	return nil
}

func TenantJwtInit(c *config.Conf, jwtGroup string) {
	jwtMap := c.Server.Http.JWT
	if jwtMap == nil {
		return
	}

	mJwt, ok := jwtMap[jwtGroup]
	if !ok {
		return
	}

	mJwt.ResetUserConfig(Tenant{})
}

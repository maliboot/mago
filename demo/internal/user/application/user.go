package application

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/maliboot/mago"

	"tests/internal/user/client"
	"tests/internal/user/client/dto"
	"tests/internal/user/client/vo"
	"tests/internal/user/domain"
)

const (
	// defaultExpire 默认令牌有效期，+8 小时用于补偿 UTC 与北京时间差。
	defaultExpire = 20 * time.Hour
	// extendExpire 勾选「延长登录」时的有效期。
	extendExpire = 5760 * time.Hour
)

// UserService 是用户上下文的用例入口，由 adapter 调用；所有方法都透传 ctx。
type UserService interface {
	Login(ctx context.Context, req *dto.UserAuthRequest) *mago.VO[vo.UserTokenVO]
	Register(ctx context.Context, req *dto.UserAuthRequest) *mago.VO[mago.EmptyVO]
	Me(ctx context.Context, userID uint) *mago.VO[vo.UserVO]
}

// UserQuery 是读模型端口，由该 DO 唯一的 repository 实现。
// 轻量 CQRS：查询走 adapter → application → repository 读方法，不加载领域聚合。
type UserQuery interface {
	ByID(ctx context.Context, id uint) (*vo.UserVO, error)
}

// userService 同时实现 adapter 用的 UserService 与跨上下文用的 api.UserService。
type userService struct {
	users   domain.UserGateway
	queries UserQuery
	hasher  PasswordHasher
	tokens  TokenIssuer
}

// NewUserService 由启动入口注入出站端口、读模型与签发能力。
func NewUserService(
	users domain.UserGateway,
	queries UserQuery,
	hasher PasswordHasher,
	tokens TokenIssuer,
) UserService {
	return &userService{
		users:   users,
		queries: queries,
		hasher:  hasher,
		tokens:  tokens,
	}
}

// FindByID 供其他上下文调用：走读模型直接返回视图，不暴露聚合。
func (s *userService) FindByID(ctx context.Context, id uint) (*vo.UserVO, error) {
	return s.queries.ByID(ctx, id)
}

func (s *userService) Login(ctx context.Context, req *dto.UserAuthRequest) *mago.VO[vo.UserTokenVO] {
	user, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		// 只有「查无此人」才转成认证失败；其他错误（数据库、网关）原样上抛，不能伪装成密码错误。
		if !isNotFound(err) {
			return mago.NewVO[vo.UserTokenVO](err, nil)
		}

		return mago.NewVO[vo.UserTokenVO](errInvalidCredentials(), nil)
	}

	if !s.hasher.Verify(ctx, req.Password, user.PasswordHash) {
		return mago.NewVO[vo.UserTokenVO](errInvalidCredentials(), nil)
	}

	// 账号状态由领域方法判定，错误码与文案由 domain 给出，这里不做二次包装。
	if err := user.CanLogin(); err != nil {
		return mago.NewVO[vo.UserTokenVO](err, nil)
	}

	expire := defaultExpire
	if req.IsExtendExpire {
		expire = extendExpire
	}

	token, expireAt, err := s.tokens.Issue(ctx, strconv.FormatUint(uint64(user.ID), 10), expire)
	if err != nil {
		return mago.NewVO[vo.UserTokenVO](err, nil)
	}

	return mago.NewVO[vo.UserTokenVO](nil, vo.NewUserTokenVO(token, expireAt.Format(time.DateTime)))
}

func (s *userService) Register(ctx context.Context, req *dto.UserAuthRequest) *mago.VO[mago.EmptyVO] {
	switch _, err := s.users.GetByEmail(ctx, req.Email); {
	case err == nil:
		return mago.NewVO[mago.EmptyVO](mago.NewError(client.ErrEmailTaken), nil)
	case !isNotFound(err):
		return mago.NewVO[mago.EmptyVO](err, nil)
	}

	hash, err := s.hasher.Hash(ctx, req.Password)
	if err != nil {
		return mago.NewVO[mago.EmptyVO](mago.NewError(mago.ErrServerError), nil)
	}

	user, err := domain.NewUser(req.Email, hash, "")
	if err != nil {
		return mago.NewVO[mago.EmptyVO](err, nil)
	}

	return mago.NewVO[mago.EmptyVO](s.users.Create(ctx, user), nil)
}

func (s *userService) Me(ctx context.Context, userID uint) *mago.VO[vo.UserVO] {
	res, err := s.FindByID(ctx, userID)

	return mago.NewVO[vo.UserVO](err, res)
}

// errInvalidCredentials 统一登录失败的对外错误，避免暴露账号是否存在。
func errInvalidCredentials() error {
	return mago.NewError(mago.ErrAuthLoginFailed).WithMsg("用户不存在或密码错误")
}

// isNotFound 判断网关返回的错误是否为「资源不存在」。
func isNotFound(err error) bool {
	var e mago.Error

	return errors.As(err, &e) && e.Code() == mago.ErrUriNotFound
}

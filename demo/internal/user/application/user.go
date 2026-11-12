package application

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/maliboot/mago"
	"github.com/maliboot/mago/dispatcher"
	"github.com/maliboot/mago/logx"

	"tests/internal/user/client"
	"tests/internal/user/client/constant"
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
	SendVerifyCode(ctx context.Context, req *dto.VerifyCodeRequest) *mago.VO[mago.EmptyVO]
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
	codes   CodeSender
	tasks   TaskQueue
}

// NewUserService 由启动入口注入出站端口、读模型与签发能力。
// 非单例：用例服务按依赖装配，由组合根创建一次并注入 adapter。
func NewUserService(
	users domain.UserGateway,
	queries UserQuery,
	hasher PasswordHasher,
	tokens TokenIssuer,
	codes CodeSender,
	tasks TaskQueue,
) UserService {
	return &userService{
		users:   users,
		queries: queries,
		hasher:  hasher,
		tokens:  tokens,
		codes:   codes,
		tasks:   tasks,
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

	if err := s.users.Create(ctx, user); err != nil {
		return mago.NewVO[mago.EmptyVO](err, nil)
	}

	s.publishRegistered(ctx, user)

	return mago.NewVO[mago.EmptyVO](nil, nil)
}

// publishRegistered 把「用户已注册」事件投给异步队列（优先 redis，README §12.3）。
// 注册已落库，投递失败只降级告警、不回滚也不阻断；需要强一致时改 Outbox（README §14.1 第 7 条）。
func (s *userService) publishRegistered(ctx context.Context, user *domain.User) {
	if s.tasks == nil {
		return
	}

	payload, err := json.Marshal(dto.UserRegisteredEvent{
		UserID: user.ID,
		Email:  user.Email,
		At:     mago.DateTime{Time: time.Now()},
	})
	if err != nil {
		logx.GroupWarnf("queue", "用户注册事件序列化失败: %v", err)
		return
	}

	if err := s.tasks.Enqueue(ctx, dispatcher.NewTask(constant.TaskUserRegistered, nil, payload)); err != nil {
		logx.GroupWarnf("queue", "用户注册事件入队失败: %v", err)
	}
}

func (s *userService) Me(ctx context.Context, userID uint) *mago.VO[vo.UserVO] {
	res, err := s.FindByID(ctx, userID)

	return mago.NewVO[vo.UserVO](err, res)
}

// SendVerifyCode 生成 6 位数字验证码并交给 CodeSender 发送。
// 演示工程不落库；真实项目应把验证码写入 Redis 并设置与邮件文案一致的有效期。
func (s *userService) SendVerifyCode(ctx context.Context, req *dto.VerifyCodeRequest) *mago.VO[mago.EmptyVO] {
	email := domain.NormalizeEmail(req.Email)
	if email == "" {
		return mago.NewVO[mago.EmptyVO](mago.NewError(mago.ErrInvalidParams).WithMsg("邮箱不能为空"), nil)
	}

	code, err := newVerifyCode()
	if err != nil {
		return mago.NewVO[mago.EmptyVO](mago.NewError(mago.ErrServerError), nil)
	}

	if err = s.codes.SendCode(ctx, email, code); err != nil {
		return mago.NewVO[mago.EmptyVO](err, nil)
	}

	return mago.NewVO[mago.EmptyVO](nil, nil)
}

// newVerifyCode 生成 6 位数字验证码，取密码学安全随机数并保前导零。
func newVerifyCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
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

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
	// DefaultTokenTTL 默认登录令牌有效期；入口层用同一策略给 JWT 中间件签发（§12.5）。
	DefaultTokenTTL = 20 * time.Hour
	// ExtendTokenTTL 勾选「延长登录」时的有效期。
	ExtendTokenTTL = 5760 * time.Hour
	// verifyCodeTTL 邮箱验证码有效期，与邮件正文里的提示保持一致。
	verifyCodeTTL = 5 * time.Minute
)

// UserService 是用户上下文的用例入口，由 adapter 调用；所有方法都透传 ctx。
type UserService interface {
	// Login 校验账密并签发令牌：非 HTTP 入口（如 MCP）用；HTTP 走 LoginByPassword。
	Login(ctx context.Context, req *dto.UserAuthRequest) *mago.VO[vo.UserTokenVO]
	// LoginByPassword 校验账密并返回登录主体；令牌由入口的 JWT 登录中间件签发（HTTP 用）。
	LoginByPassword(ctx context.Context, req *dto.UserAuthRequest) *mago.VO[vo.LoginSubjectVO]
	LoginByEmailCode(ctx context.Context, req *dto.EmailCodeLoginRequest) *mago.VO[vo.LoginSubjectVO]
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
	// verifyCodes 存验证码：发送与校验必须共用同一份存储，否则登录永远对不上。
	verifyCodes VerifyCodeStore
	tasks       TaskQueue
}

// NewUserService 由启动入口注入出站端口、读模型与签发能力。
// 非单例：用例服务按依赖装配，由组合根创建一次并注入 adapter。
func NewUserService(
	users domain.UserGateway,
	queries UserQuery,
	hasher PasswordHasher,
	tokens TokenIssuer,
	codes CodeSender,
	verifyCodes VerifyCodeStore,
	tasks TaskQueue,
) UserService {
	return &userService{
		users:       users,
		queries:     queries,
		hasher:      hasher,
		tokens:      tokens,
		codes:       codes,
		verifyCodes: verifyCodes,
		tasks:       tasks,
	}
}

// FindByID 供其他上下文调用：走读模型直接返回视图，不暴露聚合。
func (s *userService) FindByID(ctx context.Context, id uint) (*vo.UserVO, error) {
	return s.queries.ByID(ctx, id)
}

// Login 校验账密并直接签发令牌，供非 HTTP 入口使用（MCP 工具 user_login 走这里）。
// HTTP 的密码登录改走 LoginByPassword + JWTConf 登录中间件，两条路共用 passwordSubject。
func (s *userService) Login(ctx context.Context, req *dto.UserAuthRequest) *mago.VO[vo.UserTokenVO] {
	subject, err := s.passwordSubject(ctx, req.Email, req.Password)
	if err != nil {
		return mago.NewVO[vo.UserTokenVO](err, nil)
	}

	expire := DefaultTokenTTL
	if req.IsExtendExpire {
		expire = ExtendTokenTTL
	}

	token, expireAt, err := s.tokens.Issue(ctx, strconv.FormatUint(uint64(subject.UserID), 10), expire)
	if err != nil {
		return mago.NewVO[vo.UserTokenVO](err, nil)
	}

	return mago.NewVO[vo.UserTokenVO](nil, vo.NewUserTokenVO(token, expireAt.Format(time.DateTime)))
}

// LoginByPassword 只校验账密并返回登录主体，令牌由入口的 JWT 登录中间件签发。
func (s *userService) LoginByPassword(ctx context.Context, req *dto.UserAuthRequest) *mago.VO[vo.LoginSubjectVO] {
	subject, err := s.passwordSubject(ctx, req.Email, req.Password)

	return mago.NewVO[vo.LoginSubjectVO](err, subject)
}

// passwordSubject 是账密凭证的共用校验：查人 → 验密码 → 判账号状态。
func (s *userService) passwordSubject(ctx context.Context, email, password string) (*vo.LoginSubjectVO, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// 只有「查无此人」才转成认证失败；其他错误（数据库、网关）原样上抛，不能伪装成密码错误。
		if !isNotFound(err) {
			return nil, err
		}

		return nil, errInvalidCredentials()
	}

	if !s.hasher.Verify(ctx, password, user.PasswordHash) {
		return nil, errInvalidCredentials()
	}

	// 账号状态由领域方法判定，错误码与文案由 domain 给出，这里不做二次包装。
	if err := user.CanLogin(); err != nil {
		return nil, err
	}

	return loginSubjectOf(user), nil
}

// loginSubjectOf 把聚合转成登录主体。demo 固定版本号 1；真实项目取用户表 token_version，
// 改密 / 停用时 +1 让旧令牌失效（账密与验证码两条登录路共用，避免各写一份）。
func loginSubjectOf(user *domain.User) *vo.LoginSubjectVO {
	return vo.NewLoginSubjectVO(user.ID, user.Email, 1)
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

	// 先落存储再发信：用户收到的码必须和存储里一致；发信失败时这份码没人知道，
	// 等 TTL 过期自动失效（反过来先发信再存，会出现「收到的码查不到」）。
	if err = s.verifyCodes.Save(ctx, email, code, verifyCodeTTL); err != nil {
		return mago.NewVO[mago.EmptyVO](err, nil)
	}

	if err = s.codes.SendCode(ctx, email, code); err != nil {
		return mago.NewVO[mago.EmptyVO](err, nil)
	}

	return mago.NewVO[mago.EmptyVO](nil, nil)
}

// LoginByEmailCode 校验邮箱验证码并返回登录主体。
// 令牌不在这里签发：入口把主体转成 claims 交给 JWT 中间件，签发策略统一在框架（§12.5）。
func (s *userService) LoginByEmailCode(ctx context.Context, req *dto.EmailCodeLoginRequest) *mago.VO[vo.LoginSubjectVO] {
	email := domain.NormalizeEmail(req.Email)
	if email == "" || req.Code == "" {
		return mago.NewVO[vo.LoginSubjectVO](mago.NewError(mago.ErrInvalidParams).WithMsg("邮箱和验证码不能为空"), nil)
	}

	ok, err := s.verifyCodes.Consume(ctx, email, req.Code)
	if err != nil {
		return mago.NewVO[vo.LoginSubjectVO](err, nil)
	}
	if !ok {
		// 对外不区分「没发过」「过期了」「填错了」，避免被用来探测邮箱是否注册。
		return mago.NewVO[vo.LoginSubjectVO](mago.NewError(mago.ErrAuthLoginFailed).WithMsg("验证码错误或已过期"), nil)
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if !isNotFound(err) {
			return mago.NewVO[vo.LoginSubjectVO](err, nil)
		}
		// 验证码只能证明邮箱可达，不能凭空建号。
		return mago.NewVO[vo.LoginSubjectVO](mago.NewError(mago.ErrAuthLoginFailed).WithMsg("该邮箱尚未注册"), nil)
	}

	if err := user.CanLogin(); err != nil {
		return mago.NewVO[vo.LoginSubjectVO](err, nil)
	}

	return mago.NewVO[vo.LoginSubjectVO](nil, loginSubjectOf(user))
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

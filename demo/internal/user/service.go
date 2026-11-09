package user

import (
	"strconv"
	"time"

	"github.com/maliboot/mago"
	"github.com/maliboot/mago/config"
)

type Service struct {
	repo *Repository
	c    *config.Conf
}

func NewService(c *config.Conf) *Service {
	return &Service{repo: NewRepository(c), c: c}
}

func (m *Service) Login(dto *MemberTokenRequest) *mago.VO[MemberTokenResponse] {
	tenant := m.repo.GetByEmail(dto.Email)
	if tenant == nil {
		return mago.NewVO[MemberTokenResponse](mago.NewError(mago.ErrAuthUnauthorized).WithMsg("用户不存在或密码错误"), nil)
	}
	if !m.repo.CheckPasswordHash(dto.Password, tenant.PasswordHash) {
		return mago.NewVO[MemberTokenResponse](mago.NewError(mago.ErrAuthUnauthorized).WithMsg("用户不存在或密码错误."), nil)
	}

	expire := time.Hour * (8 + 12)
	if dto.IsExtendExpire {
		expire = time.Hour * (8 + 24) * 30 * 6
	}
	token, err := m.GetToken(map[string]interface{}{"id": strconv.Itoa(int(tenant.ID))}, expire)
	return mago.NewVO[MemberTokenResponse](err, token)
}

func (m *Service) Register(dto *MemberTokenRequest) *mago.VO[mago.EmptyVO] {
	existUser := m.repo.GetByEmail(dto.Email)
	if existUser != nil {
		return mago.NewVO[mago.EmptyVO](mago.NewError(mago.ErrInvalidParams).WithMsg("用户已存在"), nil)
	}

	if dto.Email == "" || dto.Password == "" {
		return mago.NewVO[mago.EmptyVO](mago.NewError(mago.ErrInvalidParams).WithMsg("用户名称/密码不能为空"), nil)
	}

	err := m.repo.CreateUser(dto.Email, dto.Password)
	return mago.NewVO[mago.EmptyVO](err, nil)
}

func (s *Service) GetToken(data any, expire time.Duration) (*MemberTokenResponse, error) {
	if expire == 0 {
		expire = time.Hour * (8 + 12) // 默认是utc时间，要加8个小时换成北京时间
	}
	jwt, ok := s.c.Server.Http.JWT["frontend"]
	if !ok {
		return nil, mago.NewError(mago.ErrServerError)
	}
	jwt.SetTimeout(expire)
	jwt.SetMaxRefresh(expire)

	token, exp, err := jwt.TokenGenerator(data)
	if err != nil {
		return nil, err
	}
	return &MemberTokenResponse{
		Token:  token,
		Expire: exp.Format(time.DateTime),
	}, nil
}

func (s *Service) Me(qry *MemberTokenRequest) *mago.VO[MemberResponse] {
	u := s.repo.Find(int(qry.UserId))
	err := s.repo.Error()
	if err != nil {
		return mago.NewVO[MemberResponse](err, nil)
	}

	return mago.NewVO[MemberResponse](nil, &MemberResponse{
		Email: u.Email,
	})
}

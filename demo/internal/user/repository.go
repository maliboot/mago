package user

import (
	"strconv"
	"strings"

	"github.com/maliboot/mago"
	"github.com/maliboot/mago/config"
	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	mago.AbstractRepository[Model]
}

func NewRepository(c *config.Conf) *Repository {
	ins := &Repository{}
	ins.Conf = c
	ins.initialize()
	return ins
}

func (r *Repository) initialize() {
	// todo ...
}

func (r *Repository) CreateUser(email, password string) error {
	hashPwd, _ := r.HashPassword(password)
	r.Create(&Model{
		Email:        email,
		PasswordHash: hashPwd,
	})

	return r.Err
}

func (r *Repository) ListById(idList []uint) ([]Model, error) {
	idStrList := make([]string, len(idList))
	for i, id := range idList {
		idStrList[i] = strconv.Itoa(int(id))
	}
	result := r.Get([][]string{{"id", "IN", strings.Join(idStrList, ",")}}, []string{"*"}, 100)
	return *result, r.Err
}

func (r *Repository) GetByEmail(email string) *Model {
	return r.First([][]string{{"email", email}}, []string{"*"})
}

// HashPassword 加密密码
func (r *Repository) HashPassword(password string) (string, error) {
	// GenerateFromPassword 生成 bcrypt 哈希，第二个参数是成本因子（cost）
	// 成本因子越高，计算越慢，安全性越高（默认推荐 10-12）
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash 验证密码
func (r *Repository) CheckPasswordHash(password, hash string) bool {
	// CompareHashAndPassword 比较密码和哈希值，返回 nil 表示匹配
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

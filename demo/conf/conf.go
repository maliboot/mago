package conf

import (
	"fmt"

	"github.com/maliboot/mago/config"
)

type Conf struct {
	Administrator struct {
		Email    string `yaml:"email"`
		Password string `yaml:"password"`
	} `yaml:"administrator"`

	// others ...
}

// New 从同一份 conf.yml 中解出业务自定义配置，由组合根（wire）装配一次后注入使用方。
// 非单例：结果由组合根持有并注入，不再维护包级全局状态（§16.8）。
func New(c *config.Conf) (*Conf, error) {
	ins := &Conf{}
	if err := c.Scan(ins); err != nil {
		return nil, fmt.Errorf("custom config decode err: %w", err)
	}

	return ins, nil
}

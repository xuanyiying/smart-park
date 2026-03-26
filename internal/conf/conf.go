package conf

import (
	"github.com/xuanyiying/smart-park/pkg/config"
)

type Config = config.Config

func LoadConfig(path string) (*Config, error) {
	return config.Load(path)
}

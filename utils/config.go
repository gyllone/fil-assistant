package utils

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	EndPoint 	string
	ApiToken 	string
	AESKey		string
	MaxFee 		string
	GasFeeCap	string
	Confidence  uint64
}

func ReadConfig(path string) (*Config, error) {
	cfg := new(Config)
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	} else {
		return cfg, nil
	}
}

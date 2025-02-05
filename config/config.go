package config

import (
	"fmt"
	"os"

	"github.com/go-micro/plugins/v4/config/encoder/toml"
	"go-micro.dev/v4/config"
	"go-micro.dev/v4/config/reader"
	"go-micro.dev/v4/config/reader/json"
	"go-micro.dev/v4/config/source"
	"go-micro.dev/v4/config/source/file"
)

type ServerConfig struct {
	GRPCListenAddress string
	HTTPListenAddress string
	ID                string
	RegistryAddress   string
	RegisterTTL       int
	RegisterInterval  int
	Name              string
	ClientTimeOut     int
	IsMaster          bool
}

func GetCfg() (config.Config, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	// load config
	enc := toml.NewEncoder()
	cfg, err := config.NewConfig(config.WithReader(json.NewReader(reader.WithEncoder(enc))))
	if err != nil {
		return nil, err
	}
	configPath := fmt.Sprintf("%s/config.toml", dir)
	err = cfg.Load(file.NewSource(
		file.WithPath(configPath),
		source.WithEncoder(enc),
	))
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

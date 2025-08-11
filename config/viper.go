package config

import (
	"context"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
)

var (
	defaultServer = "" // inject via flags
)

type Config struct {
	Servers []Server `mapstructure:"servers" toml:"servers" json:"servers"`
}

type Server struct {
	Addr string `mapstructure:"addr" toml:"addr" json:"addr"`
	Key  string `mapstructure:"key" toml:"key" json:"key"`
}

func (s *Server) Valid() bool {
	return s.Addr != ""
}

var C *Config

func LoadConfig(ctx context.Context) error {
	viper.SetConfigFile("config.toml")
	viper.AddConfigPath("/etc/remdit")
	viper.AddConfigPath("$HOME/.remdit")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	C = &Config{}
	if err := viper.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			if defaultServer == "" {
				return err
			}
			C.Servers = []Server{}
			C.Servers = append(C.Servers, Server{Addr: defaultServer})
			log.FromContext(ctx).Debug("no config file found, using default server")
			return nil
		}
		log.FromContext(ctx).Error("failed to read config", "err", err)
		os.Exit(1)
	}
	if err := viper.Unmarshal(C); err != nil {
		log.FromContext(ctx).Error("failed to unmarshal config", "err", err)
		os.Exit(1)
	}
	return nil
}

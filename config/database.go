package config

import (
	"fmt"
	"net/url"
)

type MySQLConfig struct {
	DBConfig `mapstructure:",squash"`
}

type DBConfig struct {
	Host     string `json:"host" mapstructure:"host" yaml:"host"`
	Database string `json:"database" mapstructure:"database" yaml:"database"`
	Port     int    `json:"port" mapstructure:"port" yaml:"port"`
	Username string `json:"username" mapstructure:"username" yaml:"username"`
	Password string `json:"password" mapstructure:"password" yaml:"password"`
	Options  string `json:"options" mapstructure:"options" yaml:"options"`
}

// MySQLDefaultConfig returns default config for mysql, usually use on development.
func MySQLDefaultConfig() MySQLConfig {
	return MySQLConfig{DBConfig{
		Host:     "127.0.0.1",
		Port:     3306,
		Database: "sample",
		Username: "default",
		Password: "secret",
		Options:  "?parseTime=true",
	}}
}

func (c DBConfig) DSN() string {
	options := c.Options
	if options != "" {
		if options[0] != '?' {
			options = "?" + options
		}
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s",
		c.Username,
		url.QueryEscape(c.Password),
		c.Host,
		c.Port,
		c.Database,
		options)
}

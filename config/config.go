// YOU CAN EDIT YOUR CUSTOM CONFIG HERE

package config

// Config application
type Config struct {
	MySQL MySQLConfig `json:"mysql" mapstructure:"mysql"`
	Debug bool        `json:"debug" mapstructure:"debug"`
}

// nolint:all
func loadDefaultConfig() *Config {
	c := &Config{
		MySQL: MySQLDefaultConfig(),
	}
	return c
}

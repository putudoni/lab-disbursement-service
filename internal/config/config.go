package config

type Config struct {
	AppName string
	AppEnv  string

	HTTPPort int

	Database DatabaseConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	Timezone string
}

package config

type Config struct {
	AppName string
	AppEnv  string

	HTTPPort int

	Database DatabaseConfig
	PGProvider PGProviderConfig
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

type PGProviderConfig struct {
	Mode string
	Live PGProviderCredentials
	Mock PGProviderCredentials
}

type PGProviderCredentials struct {
	APIKey    string
	APISecret string
	BaseURL   string
}

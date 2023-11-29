package config

import (
	"os"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

// SSL File Names
var CA_CERT_FILE string = "/ca-cert.pem"
var CLIENT_CERT_FILE string = "/client-cert.pem"
var CLIENT_KEY_FILE string = "/client-key.pem"

// SSL Modes
var DISABLE_SSL = "disable"
var REQUIRE_SSL = "require"

// EnvNames
var DB_SSLMode string = "DB_SSLMode"
var DB_SSL_PASS string = "DB_SSL_PASS"
var STORAGE_KIND string = "STORAGE_KIND"
var DB_PROTOCOL string = "DB_PROTOCOL"
var DB_PASS string = "DB_PASS"
var DB_USER string = "DB_USER"
var DB_HOST string = "DB_HOST"
var DB_PORT string = "DB_PORT"
var DB_NAME string = "DB_NAME"
var APP_NAME string = "APP_NAME"
var POLL_INTERVAL_SECONDS string = "POLL_INTERVAL_SECONDS"
var MAX_JOBS string = "MAX_JOBS"

// Defaults
var defaultMaxJobs int = 10000
var defaultPollInterval time.Duration = time.Minute
var defaultKind string = "postgres"
var defaultProtocol string = "postgresql"
var defaultPass string = "password"
var defaultUser string = "user"
var defaultHost string = "localhost"
var defaultPort string = "5432"
var defaultDbname string = "db1"
var defaultAppName string = "application1"
var defaultBaseDir string = "/app"
var defaultSSLMode string = DISABLE_SSL
var defaultSSLPass string = "clientpass"

type Configs struct {
	Kind         string
	Protocol     string
	Pass         string
	User         string
	Host         string
	Port         string
	Dbname       string
	SSLConfigs   SSLConfig
	AppName      string
	MaxJobs      int
	PollInterval time.Duration
}

var globalConfigs *Configs = nil

func parseMaxJobs(cfg *Configs) {
	maxJobs, err := strconv.ParseInt(os.Getenv(MAX_JOBS), 10, 64)
	if err != nil {
		log.Error().Err(err).Msgf("could not parse MAX_JOBS env defaulting to %s", defaultMaxJobs)
		globalConfigs.MaxJobs = defaultMaxJobs
	} else {
		globalConfigs.MaxJobs = int(maxJobs)
	}
}

func ParsePollInterval(cfg *Configs) {
	interval, err := strconv.ParseInt(os.Getenv(POLL_INTERVAL_SECONDS), 10, 64)
	if err != nil {
		log.Error().Err(err).Msgf("could not parse POLLING_INTERVAL_SECONDS env defaulting to %d seconds", defaultPollInterval.Seconds())
		globalConfigs.PollInterval = defaultPollInterval
	} else {
		globalConfigs.PollInterval = time.Second * time.Duration(interval)
	}
}

func getEnvOrDefault(env string, defaultValue string) string {
	if os.Getenv(env) != "" {
		return os.Getenv(env)
	}
	return defaultValue

}

// This function assumes "/app" as the dir where the application files will be within the container.
// When developing in local, we usually do not have an "/app" folder. If it doesn't exist, that means
// we are working in the host machine and we should be using this same folder structure.
func withinContainer(base string) bool {
	_, err := os.ReadDir(base)
	return err == nil
}

type SSLConfig struct {
	SSLMode     string
	CACertPath  string
	SSLCertPath string
	SSLKeyPath  string
	SSLPassword string
}

func generateLocalBasePath() string {
	base := ""
	_, currentFile, _, _ := runtime.Caller(0)
	base = path.Dir(currentFile)
	base = path.Join(base, "..", "..", "ssl")
	base = path.Clean(base)
	return base
}

func getSSLConfigs() SSLConfig {
	base := defaultBaseDir

	tlsConfigs := SSLConfig{
		// disable | require
		SSLMode: getEnvOrDefault(DB_SSLMode, defaultSSLMode),
	}
	if tlsConfigs.SSLMode == DISABLE_SSL {
		return tlsConfigs
	}
	// Assuming there wont be a "/app" folder in "/"
	// Just to be able to develop and test outside docker
	if !withinContainer(base) {
		base = generateLocalBasePath()
	}
	tlsConfigs.CACertPath = base + CA_CERT_FILE
	tlsConfigs.SSLCertPath = base + CLIENT_CERT_FILE
	tlsConfigs.SSLKeyPath = base + CLIENT_KEY_FILE
	tlsConfigs.SSLPassword = getEnvOrDefault(DB_SSL_PASS, defaultSSLPass)

	return tlsConfigs
}

func FromEnvs() Configs {
	if globalConfigs == nil {
		globalConfigs = &Configs{
			Kind:         getEnvOrDefault(STORAGE_KIND, defaultKind),
			Protocol:     getEnvOrDefault(DB_PROTOCOL, defaultProtocol),
			Pass:         getEnvOrDefault(DB_PASS, defaultPass),
			User:         getEnvOrDefault(DB_USER, defaultUser),
			Host:         getEnvOrDefault(DB_HOST, defaultHost),
			Port:         getEnvOrDefault(DB_PORT, defaultPort),
			Dbname:       getEnvOrDefault(DB_NAME, defaultDbname),
			AppName:      getEnvOrDefault(APP_NAME, defaultAppName),
			SSLConfigs:   getSSLConfigs(),
			MaxJobs:      defaultMaxJobs,
			PollInterval: defaultPollInterval,
		}
	}
	return *globalConfigs
}

func MaxJobs() int {
	if globalConfigs == nil {
		return FromEnvs().MaxJobs
	}
	return globalConfigs.MaxJobs
}

func AppName() string {
	if globalConfigs == nil {
		return FromEnvs().AppName
	}
	return globalConfigs.AppName
}

func PollingInterval() time.Duration {
	if globalConfigs == nil {
		return FromEnvs().PollInterval
	}
	return globalConfigs.PollInterval
}

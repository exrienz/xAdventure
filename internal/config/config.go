package config

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	DBPath          string
	OpenAIBase      string
	OpenAIKey       string
	OpenAIModel     string
	KidsLLMBase     string
	KidsLLMKey      string
	KidsLLMModel    string
	ImageRouterBase string
	ImageRouterKey  string
	ImageRouterModel string
	LLMTimeoutSec    int
	LLMMaxRetries    int
	LLMTemperature   float64
	LLMTopP          float64
	RequestMaxBytes int64
	FeatureFlags    map[string]bool
	FeatureRollout  map[string]int // percentage 0-100 for gradual rollouts
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	kidsModel := getEnv("KIDS_LLM_MODEL", "")
	if kidsModel != "" && !isValidModelName(kidsModel) {
		kidsModel = "" // ignore invalid
	}

	cfg := &Config{
		Port:             getEnv("PORT", "8001"),
		DBPath:           getEnv("DB_PATH", "./adventure.db"),
		OpenAIBase:       getEnv("OPENAI_API_BASE", "https://api.openai.com/v1"),
		OpenAIKey:        getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:      getEnv("OPENAI_MODEL", "gpt-4o"),
		KidsLLMBase:      getEnv("KIDS_LLM_API_BASE", ""),
		KidsLLMKey:       getEnv("KIDS_LLM_API_KEY", ""),
		KidsLLMModel:     kidsModel,
		ImageRouterBase:  getEnv("IMAGEROUTER_API_BASE", "https://api.imagerouter.io/v1/openai"),
		ImageRouterKey:   getEnv("IMAGEROUTER_API_KEY", ""),
		ImageRouterModel: getEnv("IMAGEROUTER_MODEL", "imagerouter/auto-image"),
		LLMTimeoutSec:    getEnvInt("LLM_TIMEOUT_SEC", 60),
		LLMMaxRetries:    getEnvInt("LLM_MAX_RETRIES", 2),
		LLMTemperature:   getEnvFloat("LLM_TEMPERATURE", 0.85),
		LLMTopP:          getEnvFloat("LLM_TOP_P", 0.95),
		RequestMaxBytes:  int64(getEnvInt("REQUEST_MAX_BYTES", 1024*1024)),
		FeatureFlags:     parseFeatureFlags(getEnv("FEATURE_FLAGS", "")),
		FeatureRollout:   parseFeatureRollout(getEnv("FEATURE_ROLLOUT", "")),
	}

	return cfg, nil
}

func (c *Config) HasKidsLLMProvider() bool {
	return strings.TrimSpace(c.KidsLLMBase) != "" &&
		strings.TrimSpace(c.KidsLLMKey) != "" &&
		strings.TrimSpace(c.KidsLLMModel) != ""
}

func (c *Config) HasImageRouterProvider() bool {
	return strings.TrimSpace(c.ImageRouterBase) != "" &&
		strings.TrimSpace(c.ImageRouterKey) != "" &&
		strings.TrimSpace(c.ImageRouterModel) != ""
}

func (c *Config) FeatureEnabled(name string) bool {
	if c.FeatureFlags[name] {
		return true
	}
	pct := c.FeatureRollout[name]
	if pct <= 0 {
		return false
	}
	if pct >= 100 {
		return true
	}
	// Deterministic but simple rollout based on current second.
	// For production, replace with a stable user/session hash.
	return (time.Now().Unix() % 100) < int64(pct)
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func parseFeatureFlags(s string) map[string]bool {
	flags := make(map[string]bool)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// support both "flag" and "flag=true" / "flag=false"
		kv := strings.SplitN(part, "=", 2)
		key := kv[0]
		val := true
		if len(kv) == 2 {
			if b, err := strconv.ParseBool(kv[1]); err == nil {
				val = b
			}
		}
		flags[key] = val
	}
	return flags
}

func parseFeatureRollout(s string) map[string]int {
	rollout := make(map[string]int)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		key := kv[0]
		val := 100
		if len(kv) == 2 {
			if i, err := strconv.Atoi(kv[1]); err == nil {
				val = i
				if val < 0 {
					val = 0
				}
				if val > 100 {
					val = 100
				}
			}
		}
		rollout[key] = val
	}
	return rollout
}

var modelNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._:/-]+$`)

func isValidModelName(name string) bool {
	if name == "" || strings.Contains(name, "..") || strings.HasPrefix(name, "/") {
		return false
	}
	return modelNameRegex.MatchString(name)
}

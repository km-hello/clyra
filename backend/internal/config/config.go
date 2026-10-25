package config

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"clyra/backend/internal/textutil"
)

type Config struct {
	Addr, StaticDir, AppOrigin, SessionPassword, DatabasePath, TelegramBotToken, SerpAPIKey string
	Production                                                                              bool
}

func Load() (Config, error) {
	c := Config{Addr: env("CLYRA_ADDR", "127.0.0.1:8080"), StaticDir: env("CLYRA_STATIC_DIR", "./frontend/dist"), AppOrigin: os.Getenv("CLYRA_APP_ORIGIN"), SessionPassword: os.Getenv("CLYRA_SESSION_PASSWORD"), DatabasePath: env("CLYRA_DATABASE_PATH", "./data/clyra.db"), TelegramBotToken: os.Getenv("CLYRA_TELEGRAM_BOT_TOKEN"), SerpAPIKey: os.Getenv("CLYRA_SERP_API_KEY")}
	mode := env("CLYRA_ENV", "development")
	if mode != "development" && mode != "production" {
		return c, fmt.Errorf("CLYRA_ENV must be development or production")
	}
	c.Production = mode == "production"
	if c.AppOrigin == "" && !c.Production {
		c.AppOrigin = "http://localhost:5173"
	}
	err := c.Validate()
	return c, err
}
func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
func (c *Config) Validate() error {
	u, err := url.Parse(c.AppOrigin)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(c.AppOrigin, "#") || u.Path != "" || u.Opaque != "" || (u.Scheme != "http" && u.Scheme != "https") || c.Production && u.Scheme != "https" {
		return fmt.Errorf("CLYRA_APP_ORIGIN must be an exact HTTP origin (HTTPS in production)")
	}
	host := strings.ToLower(u.Hostname())
	if addr, err := netip.ParseAddr(host); err == nil {
		if addr.Zone() != "" || addr.Is4In6() {
			return fmt.Errorf("CLYRA_APP_ORIGIN must use an unscoped IP address")
		}
		host = addr.String()
	} else if !regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?\.?$`).MatchString(host) || regexp.MustCompile(`(?:^|\.)(?:[0-9]+|0x[0-9a-f]+)\.?$`).MatchString(host) {
		return fmt.Errorf("CLYRA_APP_ORIGIN must use an ASCII hostname or a standard IP address")
	}
	port := u.Port()
	if strings.HasSuffix(u.Host, ":") {
		return fmt.Errorf("CLYRA_APP_ORIGIN must not contain an empty port")
	}
	if port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return fmt.Errorf("CLYRA_APP_ORIGIN port must be between 1 and 65535")
		}
		port = strconv.Itoa(number)
		if u.Scheme == "https" && port == "443" || u.Scheme == "http" && port == "80" {
			port = ""
		}
	}
	if port != "" {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	c.AppOrigin = u.Scheme + "://" + host
	chars := map[rune]bool{}
	for _, r := range c.SessionPassword {
		chars[r] = true
	}
	if textutil.Length(c.SessionPassword) < 32 || len(chars) < 8 || regexp.MustCompile(`(?i)replace|example|placeholder|change.?me|your.?secret`).MatchString(c.SessionPassword) {
		return fmt.Errorf("CLYRA_SESSION_PASSWORD must be a random secret of at least 32 characters")
	}
	if strings.TrimSpace(c.DatabasePath) == "" || strings.ContainsRune(c.DatabasePath, 0) || c.DatabasePath == ":memory:" || strings.HasPrefix(c.DatabasePath, "file:") {
		return fmt.Errorf("CLYRA_DATABASE_PATH must be a persistent local file")
	}
	return nil
}

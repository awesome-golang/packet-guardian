// This source file is part of the Packet Guardian project.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

// Config defines the configuration struct for the application
type Config struct {
	sourceFile string
	Core       struct {
		SiteTitle          string
		SiteCompanyName    string
		SiteDomainName     string
		SiteFooterText     string
		JobSchedulerWakeUp string
	}
	Logging struct {
		Enabled    bool
		EnableHTTP bool
		Level      string
		Path       string
	}
	Database struct {
		Type         string
		Address      string
		Port         int
		Username     string
		Password     string
		Name         string
		Retry        int
		RetryTimeout string
	}
	Registration struct {
		RegistrationPolicyFile      string
		AllowManualRegistrations    bool
		DefaultDeviceLimit          int
		DefaultDeviceExpirationType string
		RollingExpirationLength     string
		DefaultDeviceExpiration     string
		ManualRegPlatforms          []string
	}
	Guest struct {
		Enabled              bool
		GuestOnly            bool
		DeviceLimit          int
		DeviceExpirationType string
		DeviceExpiration     string
		Checker              string
		VerifyCodeExpiration int
		DisableCaptcha       bool
		RegPageHeader        string

		Email struct {
		}

		Twilio struct {
			AccountSID  string
			AuthToken   string
			PhoneNumber string
		}

		Smseagle struct {
			Address      string
			Username     string
			Password     string
			HighPriority int
			FlashMsg     int
		}
	}
	Webserver struct {
		Address             string
		HttpPort            int
		HttpsPort           int
		TLSCertFile         string
		TLSKeyFile          string
		RedirectHttpToHttps bool
		SessionStore        string
		SessionName         string
		SessionsDir         string
		SessionsAuthKey     string
		SessionsEncryptKey  string
	}
	Auth struct {
		AuthMethod        []string
		AdminUsers        []string
		HelpDeskUsers     []string
		ReadOnlyUsers     []string
		APIReadOnlyUsers  []string
		APIReadWriteUsers []string
		APIStatusUsers    []string

		LDAP struct {
			Server             string
			Port               int
			UseSSL             bool
			InsecureSkipVerify bool
			SkipTLS            bool
			DomainName         string
		}
		Radius struct {
			Servers []string
			Port    int
			Secret  string
		}
		CAS struct {
			Server     string
			ServiceURL string
		}
	}
	DHCP struct {
		ConfigFile string
	}
	Email struct {
		Address     string
		Port        int
		Username    string
		Password    string
		FromAddress string
		ToAddresses []string
	}
}

func FindConfigFile() string {
	if os.Getenv("PG_CONFIG") != "" && FileExists(os.Getenv("PG_CONFIG")) {
		return os.Getenv("PG_CONFIG")
	} else if FileExists("./config.toml") {
		return "./config.toml"
	} else if FileExists("./config/config.toml") {
		return "./config/config.toml"
	} else if FileExists(os.ExpandEnv("$HOME/.pg/config.toml")) {
		return os.ExpandEnv("$HOME/.pg/config.toml")
	} else if FileExists("/etc/packet-guardian/config.toml") {
		return "/etc/packet-guardian/config.toml"
	}
	return ""
}

func NewEmptyConfig() *Config {
	return &Config{}
}

func NewConfig(configFile string) (conf *Config, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Unknown panic")
			}
		}
	}()

	if configFile == "" {
		configFile = "config.toml"
	}

	var con Config
	if _, err := toml.DecodeFile(configFile, &con); err != nil {
		return nil, err
	}
	con.sourceFile = configFile
	return setSensibleDefaults(&con)
}

func setSensibleDefaults(c *Config) (*Config, error) {
	// Anything not set here implies its zero value is the default

	// Core
	c.Core.SiteTitle = setStringOrDefault(c.Core.SiteTitle, "Packet Guardian")
	c.Core.SiteFooterText = setStringOrDefault(c.Core.SiteFooterText, "The Guardian of Packets")
	c.Core.JobSchedulerWakeUp = setStringOrDefault(c.Core.JobSchedulerWakeUp, "1h")
	if _, err := time.ParseDuration(c.Core.JobSchedulerWakeUp); err != nil {
		c.Core.JobSchedulerWakeUp = "1h"
	}

	// Logging
	c.Logging.Level = setStringOrDefault(c.Logging.Level, "notice")
	c.Logging.Path = setStringOrDefault(c.Logging.Path, "logs/pg.log")

	// Database
	c.Database.Type = setStringOrDefault(c.Database.Type, "mysql")
	c.Database.Address = setStringOrDefault(c.Database.Address, "localhost")
	c.Database.RetryTimeout = setStringOrDefault(c.Database.RetryTimeout, "10s")

	// Registration
	c.Registration.RegistrationPolicyFile = setStringOrDefault(c.Registration.RegistrationPolicyFile, "config/policy.txt")
	c.Registration.DefaultDeviceExpirationType = setStringOrDefault(c.Registration.DefaultDeviceExpirationType, "rolling")
	c.Registration.RollingExpirationLength = setStringOrDefault(c.Registration.RollingExpirationLength, "4380h")
	if _, err := time.ParseDuration(c.Registration.RollingExpirationLength); err != nil {
		c.Registration.RollingExpirationLength = "4380h"
	}

	// Guest registrations
	c.Guest.DeviceExpirationType = setStringOrDefault(c.Guest.DeviceExpirationType, "daily")
	c.Guest.DeviceExpiration = setStringOrDefault(c.Guest.DeviceExpiration, "24:00")
	c.Guest.Checker = setStringOrDefault(c.Guest.Checker, "email")
	c.Guest.VerifyCodeExpiration = setIntOrDefault(c.Guest.VerifyCodeExpiration, 3)

	// Webserver
	c.Webserver.HttpPort = setIntOrDefault(c.Webserver.HttpPort, 80)
	c.Webserver.HttpsPort = setIntOrDefault(c.Webserver.HttpsPort, 443)
	c.Webserver.SessionName = setStringOrDefault(c.Webserver.SessionName, "packet-guardian")
	c.Webserver.SessionsDir = setStringOrDefault(c.Webserver.SessionsDir, "sessions")
	c.Webserver.SessionStore = setStringOrDefault(c.Webserver.SessionStore, "filesystem")

	// Authentication
	if len(c.Auth.AuthMethod) == 0 {
		c.Auth.AuthMethod = []string{"local"}
	}

	if len(c.Auth.AdminUsers) > 0 {
		fmt.Println("Setting Auth.AdminUsers is deprecated and no longer used")
	}
	if len(c.Auth.HelpDeskUsers) > 0 {
		fmt.Println("Setting Auth.HelpDeskUsers is deprecated and no longer used")
	}
	if len(c.Auth.ReadOnlyUsers) > 0 {
		fmt.Println("Setting Auth.ReadOnlyUsers is deprecated and no longer used")
	}
	if len(c.Auth.APIReadOnlyUsers) > 0 {
		fmt.Println("Setting Auth.APIReadOnlyUsers is deprecated and no longer used")
	}
	if len(c.Auth.APIReadWriteUsers) > 0 {
		fmt.Println("Setting Auth.APIReadWriteUsers is deprecated and no longer used")
	}
	if len(c.Auth.APIStatusUsers) > 0 {
		fmt.Println("Setting Auth.APIStatusUsers is deprecated and no longer used")
	}

	c.Auth.LDAP.Server = setStringOrDefault(c.Auth.LDAP.Server, "127.0.0.1")
	c.Auth.LDAP.Port = setIntOrDefault(c.Auth.LDAP.Port, 389)

	// DHCP
	c.DHCP.ConfigFile = setStringOrDefault(c.DHCP.ConfigFile, "config/dhcp.conf")

	// Email
	if c.Email.Address != "" {
		if c.Email.FromAddress == "" {
			return nil, errors.New("Email.FromAddress cannot be empty")
		}
		if len(c.Email.ToAddresses) == 0 {
			return nil, errors.New("Email.ToAddresses cannot be empty")
		}
		c.Email.Port = setIntOrDefault(c.Email.Port, 25)
	}
	return c, nil
}

// Given string s, if it is empty, return v else return s.
func setStringOrDefault(s, v string) string {
	if s == "" {
		return v
	}
	return s
}

// Given integer s, if it is 0, return v else return s.
func setIntOrDefault(s, v int) int {
	if s == 0 {
		return v
	}
	return s
}

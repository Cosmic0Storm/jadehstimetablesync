package internal

import (
	"net/url"
	"time"

	"github.com/spf13/viper"
)

const (
	CONFIG_KEY_DEGREE               = "Degree"
	CONFIG_KEY_MAX_KW_OFFSET        = "MaxKWOffset"
	CONFIG_KEY_MIN_SEMESTER         = "MinSemester"
	CONFIG_KEY_MAX_SEMESTER         = "MaxSemester"
	CONFIG_KEY_CRONSCHEDULE         = "CronSchedule"
	CONFIG_KEY_MODULE_WHITELIST     = "ModuleWhitelist"
	CONFIG_KEY_CALENDAR_URL         = "CalendarUrl"
	ENVIRONMENT_KEY_CALDAV_USERNAME = "calendar_username"
	ENVIRONMENT_KEY_CALDAV_PASSWORD = "calendar_password"

	DEFAULT_CALENDAR_URL  = "https://example.com/caldavurl"
	DEFAULT_TIMEZONE      = "Europe/Berlin"
	DEFAULT_DEGREE        = "MIT Winf"
	DEFAULT_MAX_KW_OFFSET = 4
	DEFAULT_MIN_SEMESTER  = 1
	DEFAULT_MAX_SEMESTER  = 1
	DEFAULT_CRONSCHEDULE  = "0 0,8,14,22 * * *"

	DEFAULT_CONFIG_FILE = "config.yml"
	DEFAULT_CONFIG_PATH = "."
)

const (
	ERROR_ONLY_DEFAULT_CALENDAR_URL_FOUND = "No CalendarUrl in config specified"
	ERROR_PARSE_CALENDAR_URL              = "Could not parse CalendarUrl"
)

type RawConfig struct {
	CalendarUrl     string
	Degree          string
	MaxKWOffset     int
	MinSemester     int
	MaxSemester     int
	CronSchedule    string
	Timezone        string
	ModuleWhitelist []string
}

func (c *RawConfig) GetConfig() (*Config, error) {

	url, err := url.Parse(c.CalendarUrl)
	if err != nil {
		return nil, err
	}

	timezone, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return nil, err
	}

	return &Config{
		CalendarUrl:     *url,
		Degree:          c.Degree,
		MaxKWOffset:     c.MaxKWOffset,
		MinSemester:     c.MinSemester,
		MaxSemester:     c.MaxSemester,
		Timezone:        *timezone,
		CronSchedule:    c.CronSchedule,
		ModuleWhitelist: c.ModuleWhitelist,
	}, nil

}

type Config struct {
	CalendarUrl     url.URL
	Degree          string
	MaxKWOffset     int
	MinSemester     int
	MaxSemester     int
	CronSchedule    string
	Timezone        time.Location
	ModuleWhitelist []string
}

func SetupConfig() (*Config, error) {

	setDefaults()

	viper.SetConfigFile(DEFAULT_CONFIG_FILE)
	viper.AddConfigPath(DEFAULT_CONFIG_PATH)

	bindEnvironmentVariables()

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	rc := &RawConfig{}
	err = viper.Unmarshal(rc)
	if err != nil {
		return nil, err
	}

	c, err := rc.GetConfig()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func setDefaults() {
	viper.SetDefault(CONFIG_KEY_DEGREE, DEFAULT_DEGREE)
	viper.SetDefault(CONFIG_KEY_MAX_KW_OFFSET, DEFAULT_MAX_KW_OFFSET)
	viper.SetDefault(CONFIG_KEY_MIN_SEMESTER, DEFAULT_MIN_SEMESTER)
	viper.SetDefault(CONFIG_KEY_MAX_SEMESTER, DEFAULT_MAX_SEMESTER)
	viper.SetDefault(CONFIG_KEY_CRONSCHEDULE, DEFAULT_CRONSCHEDULE)
	viper.SetDefault(CONFIG_KEY_MODULE_WHITELIST, []string{})
	viper.SetDefault(CONFIG_KEY_CALENDAR_URL, DEFAULT_CALENDAR_URL)
}

func bindEnvironmentVariables() {
	viper.BindEnv(ENVIRONMENT_KEY_CALDAV_USERNAME)
	viper.BindEnv(ENVIRONMENT_KEY_CALDAV_PASSWORD)
}

func GetCaldavConfiguration(conf *Config) CaldavConfiguration {
	return CaldavConfiguration{
		Username:    viper.GetString(ENVIRONMENT_KEY_CALDAV_USERNAME),
		Password:    viper.GetString(ENVIRONMENT_KEY_CALDAV_PASSWORD),
		CalendarURL: conf.CalendarUrl,
	}
}

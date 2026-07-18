package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	GoogleLink         string
	PartySize          int
	DateRangeFrom      time.Time
	DateRangeTo        time.Time
	DayPreference      []int
	CheckIntervalMins  int
	AutoBook           bool
	LogLevel           string
}

func LoadConfig() *Config {
	cfg := &Config{
		GoogleLink:        os.Getenv("RESTAURANT_GOOGLE_LINK"),
		PartySize:         getInt("RESTAURANT_PARTY_SIZE", 2),
		CheckIntervalMins: getInt("CHECK_INTERVAL_MINUTES", 5),
		AutoBook:          getBool("AUTO_BOOK", true),
		LogLevel:          os.Getenv("LOG_LEVEL"),
	}

	// Parse dates
	if dateFrom := os.Getenv("DATE_RANGE_FROM"); dateFrom != "" {
		cfg.DateRangeFrom, _ = time.Parse("2006-01-02", dateFrom)
	} else {
		cfg.DateRangeFrom = time.Now()
	}

	if dateTo := os.Getenv("DATE_RANGE_TO"); dateTo != "" {
		cfg.DateRangeTo, _ = time.Parse("2006-01-02", dateTo)
	} else {
		cfg.DateRangeTo = time.Now().AddDate(0, 0, 30)
	}

	// Parse day preferences
	if dayStr := os.Getenv("DAY_PREFERENCE"); dayStr != "" {
		for _, d := range strings.Split(dayStr, ",") {
			if day, err := strconv.Atoi(strings.TrimSpace(d)); err == nil {
				cfg.DayPreference = append(cfg.DayPreference, day)
			}
		}
	} else {
		cfg.DayPreference = []int{5, 6} // Default: Friday, Saturday
	}

	return cfg
}

func getInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return strings.ToLower(val) == "true"
	}
	return defaultVal
}

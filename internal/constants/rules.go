package constants

import (
	"github.com/kerimovok/go-pkg-utils/config"
	"github.com/kerimovok/go-pkg-utils/validator"
)

var EnvValidationRules = []validator.ValidationRule{
	// Server validation
	{
		Variable: "PORT",
		Default:  "3003",
		Rule:     config.IsValidPort,
		Message:  "server port is required and must be a valid port number",
	},
	{
		Variable: "GO_ENV",
		Default:  "development",
		Rule:     func(v string) bool { return v == "development" || v == "production" },
		Message:  "GO_ENV must be either 'development' or 'production'",
	},

	// Database validation
	{
		Variable: "DB_HOST",
		Rule:     func(v string) bool { return v != "" },
		Message:  "database host is required",
	},
	{
		Variable: "DB_PORT",
		Default:  "5432",
		Rule:     config.IsValidPort,
		Message:  "database port is required and must be a valid port number",
	},
	{
		Variable: "DB_USER",
		Rule:     func(v string) bool { return v != "" },
		Message:  "database user is required",
	},
	{
		Variable: "DB_NAME",
		Default:  "storage",
		Rule:     func(v string) bool { return v != "" },
		Message:  "database name is required",
	},
}

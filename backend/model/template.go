package model

import (
	"regexp"
	"time"
)

type TemplateEntry struct {
	Key          string `json:"key"`
	DefaultValue string `json:"defaultValue"`
	ValueType    string `json:"valueType"`
	Required     bool   `json:"required"`
	IsSensitive  bool   `json:"isSensitive"`
	Description  string `json:"description"`
}

type TemplateVariable struct {
	Name         string `json:"name"`
	DefaultValue string `json:"defaultValue"`
	Required     bool   `json:"required"`
	IsSensitive  bool   `json:"isSensitive"`
	Description  string `json:"description"`
}

type Template struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Format      string             `json:"format"`
	Body        string             `json:"body"`
	Variables   []TemplateVariable `json:"variables"`
	Entries     []TemplateEntry    `json:"entries"`
	OwnerUserID string             `json:"ownerUserId,omitempty"`
	IsCustom    bool               `json:"isCustom"`
	CreatedAt   time.Time          `json:"createdAt,omitempty"`
	UpdatedAt   time.Time          `json:"updatedAt,omitempty"`
}

func BaseTemplate() Template {
	return GlobalLoggingTemplate()
}

func GlobalLoggingTemplate() Template {
	body := `logging:
  level:
    root: ${LOG_LEVEL_ROOT}
    com.company.core: ${LOG_LEVEL_COMPANY}
  pattern:
    console: "%d{yyyy-MM-dd HH:mm:ss.SSS} [%thread] %-5level %logger{36} - [TraceID: %X{traceId}] - %msg%n"
    file: "%d{yyyy-MM-dd HH:mm:ss.SSS} [%thread] %-5level %logger{36} - [TraceID: %X{traceId}] - %msg%n"
  file:
    name: /var/log/apps/${APP_NAME}/application.log
    max-history: 30
    max-size: 10MB
`
	variables := ExtractTemplateVariables(body)
	for index := range variables {
		switch variables[index].Name {
		case "LOG_LEVEL_ROOT":
			variables[index].DefaultValue = "INFO"
			variables[index].Description = "Root log level; use DEBUG for dev and INFO for prod"
		case "LOG_LEVEL_COMPANY":
			variables[index].DefaultValue = "INFO"
			variables[index].Description = "Company core package log level"
		case "APP_NAME":
			variables[index].Description = "Application name used in the log file path"
		}
	}
	return Template{
		ID:          "global-logging-template",
		Name:        "Global Logging Template",
		Description: "Enterprise logging baseline with standard console/file patterns and rolling file settings.",
		Format:      "yaml",
		Body:        body,
		Variables:   variables,
	}
}

func InfrastructureTemplates() []Template {
	base := BaseTemplate()
	return []Template{base, SpringBootBaseTemplate()}
}

func SpringBootBaseTemplate() Template {
	body := `server:
  port: ${APP_PORT}
  tomcat:
    max-threads: 200
spring:
  datasource:
    url: jdbc:mysql://${DB_HOST}:3306/${DB_NAME}?useSSL=false
    username: ${DB_USER}
    password: ${DB_SECRET}
  redis:
    host: ${REDIS_HOST}
management:
  endpoints:
    web:
      exposure:
        include: 'health,info,metrics'
`
	variables := ExtractTemplateVariables(body)
	for index := range variables {
		switch variables[index].Name {
		case "APP_PORT":
			variables[index].DefaultValue = "8080"
			variables[index].Description = "Application listen port"
		case "DB_HOST":
			variables[index].Description = "Database host"
		case "DB_NAME":
			variables[index].Description = "Database schema/name"
		case "DB_USER":
			variables[index].Description = "Database username"
		case "DB_SECRET":
			variables[index].IsSensitive = true
			variables[index].Description = "Database password or secret reference"
		case "REDIS_HOST":
			variables[index].Description = "Redis host"
		}
	}
	return Template{
		ID:          "spring-boot-base-template",
		Name:        "Spring-Boot-Standard-Template",
		Description: "Infrastructure skeleton for Spring Boot services with project-specific variables.",
		Format:      "yaml",
		Body:        body,
		Variables:   variables,
	}
}

func ExtractTemplateVariables(body string) []TemplateVariable {
	matches := regexp.MustCompile(`\$\{([A-Z0-9_]+)\}`).FindAllStringSubmatch(body, -1)
	seen := map[string]bool{}
	variables := make([]TemplateVariable, 0)
	for _, match := range matches {
		if len(match) < 2 || seen[match[1]] {
			continue
		}
		seen[match[1]] = true
		variables = append(variables, TemplateVariable{
			Name:     match[1],
			Required: true,
		})
	}
	return variables
}

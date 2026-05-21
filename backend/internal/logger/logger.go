package logger

import (
	"io"
	"log/slog"
	"os"
)

const (
	FieldService         = "service"
	FieldCategory        = "category"
	FieldProjectID       = "project_id"
	FieldEnvironment     = "environment"
	FieldConfigID        = "config_id"
	FieldConfigKey       = "config_key"
	FieldReviewRequestID = "review_request_id"
	FieldActor           = "actor"
	FieldUserID          = "user_id"
	FieldRole            = "role"
)

const serviceName = "config-man"

var (
	Log        *slog.Logger
	Main       *slog.Logger
	API        *slog.Logger
	Processor  *slog.Logger
	Store      *slog.Logger
	DB         *slog.Logger
	Auth       *slog.Logger
	Project    *slog.Logger
	Config   *slog.Logger
	Review   *slog.Logger
	Template *slog.Logger
)

func init() {
	Init(os.Stdout)
}

func Init(writer io.Writer) {
	Log = slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	Main = category("main")
	API = category("api")
	Processor = category("processor")
	Store = category("store")
	DB = category("db")
	Auth = category("auth")
	Project = category("project")
	Config = category("config")
	Review = category("review")
	Template = category("template")
}

func New() *slog.Logger {
	return Main
}

func category(name string) *slog.Logger {
	return Log.With(
		slog.String(FieldService, serviceName),
		slog.String(FieldCategory, name),
	)
}

package logger

import (
	"io"
	"log/slog"
	"os"
)

const (
	FieldService         = "service"
	FieldLayer           = "layer"
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
	Log           *slog.Logger
	MainLog       *slog.Logger
	APILog        *slog.Logger
	ProcessorLog  *slog.Logger
	StoreLog      *slog.Logger
	DBLog         *slog.Logger
	AuthLog       *slog.Logger
	ProjectLog    *slog.Logger
	ConfigLog     *slog.Logger
	ReviewLog     *slog.Logger
	ValidationLog *slog.Logger
)

func init() {
	Init(os.Stdout)
}

func Init(writer io.Writer) {
	Log = slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	MainLog = layer("main")
	APILog = layer("api")
	ProcessorLog = layer("processor")
	StoreLog = layer("store")
	DBLog = layer("db")
	AuthLog = layer("auth")
	ProjectLog = layer("project")
	ConfigLog = layer("config")
	ReviewLog = layer("review")
	ValidationLog = layer("validation")
}

func New() *slog.Logger {
	return MainLog
}

func layer(name string) *slog.Logger {
	return Log.With(
		slog.String(FieldService, serviceName),
		slog.String(FieldLayer, name),
	)
}

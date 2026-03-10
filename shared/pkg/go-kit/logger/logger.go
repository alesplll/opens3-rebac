package logger

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/contextx"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/contextx/claimsctx"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/contextx/ipctx"
	traceidctx "github.com/alesplll/opens3-rebac/shared/pkg/go-kit/contextx/traceIDctx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otelLog "go.opentelemetry.io/otel/log"
	otelSdkLog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Global singleton logger
var (
	globalLogger *logger
	initOnce     sync.Once
	dynamicLevel zap.AtomicLevel
	otelProvider *otelSdkLog.LoggerProvider // OTLP provider for graceful shutdown

	cfg LoggerConfig
)

type logger struct {
	zapLogger *zap.Logger
}

func Init(loggerConfig LoggerConfig) error {
	initOnce.Do(func() {
		cfg = loggerConfig
		dynamicLevel = zap.NewAtomicLevelAt(parseLevel(cfg.LogLevel()))

		cores := buildCores(cfg.EnableOLTP())
		tee := zapcore.NewTee(cores...)

		zapLogger := zap.New(tee, zap.AddCaller(), zap.AddCallerSkip(2))

		globalLogger = &logger{
			zapLogger: zapLogger,
		}
	})

	return nil
}

func buildCores(enableOTLP bool) []zapcore.Core {
	cores := []zapcore.Core{
		createStdoutCore(),
	}

	if enableOTLP {
		if otlpCore := createOTLPCore(); otlpCore != nil {
			cores = append(cores, otlpCore)
		}
	}

	return cores
}

func createStdoutCore() zapcore.Core {
	encoderCfg := buildProductionEncoderConfig()
	var encoder zapcore.Encoder
	if cfg.AsJSON() {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	return zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), dynamicLevel)
}

func createOTLPCore() *SimpleOTLPCore {
	otlpLogger, err := createOTLPLogger(cfg.OTLPEndpoint())
	if err != nil {
		return nil
	}

	return NewSimpleOTLPCore(otlpLogger, dynamicLevel)
}

func createOTLPLogger(endpoint string) (otelLog.Logger, error) {
	ctx := context.Background()

	exporter, err := createOTLPExporter(ctx, cfg.OTLPEndpoint())
	if err != nil {
		return nil, err
	}

	resource, err := createResource(ctx)
	if err != nil {
		return nil, err
	}

	provider := otelSdkLog.NewLoggerProvider(
		otelSdkLog.WithResource(resource),
		otelSdkLog.WithProcessor(otelSdkLog.NewBatchProcessor(exporter)),
	)

	return provider.Logger(fmt.Sprintf("logger:%s", cfg.ServiceName())), nil
}

func createOTLPExporter(ctx context.Context, endpoint string) (*otlploggrpc.Exporter, error) {
	return otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(endpoint),
		otlploggrpc.WithInsecure(),
	)
}

func createResource(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName()),
			attribute.String("deployment.environment", cfg.ServiceEnvironment()),
		),
	)
}

func buildProductionEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
}

// Logger() return global enrich-aware logger
func Logger() *logger {
	return globalLogger
}

func SetNopLogger() {
	globalLogger = &logger{
		zapLogger: zap.NewNop(),
	}
}

func Sync() error {
	if globalLogger != nil {
		return globalLogger.zapLogger.Sync()
	}

	return nil
}

func With(fields ...zap.Field) *logger {
	if globalLogger == nil {
		return &logger{zapLogger: zap.NewNop()}
	}

	return &logger{
		zapLogger: globalLogger.zapLogger.With(fields...),
	}
}

func WithContext(ctx context.Context) *logger {
	if globalLogger == nil {
		return &logger{zapLogger: zap.NewNop()}
	}

	return &logger{
		zapLogger: globalLogger.zapLogger.With(fieldsFromContext(ctx)...),
	}
}

// Debug enrich-aware debug log
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	globalLogger.Debug(ctx, msg, fields...)
}

// Info enrich-aware info log
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	globalLogger.Info(ctx, msg, fields...)
}

// Warn enrich-aware warn log
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	globalLogger.Warn(ctx, msg, fields...)
}

// Error enrich-aware error log
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	globalLogger.Error(ctx, msg, fields...)
}

// Fatal enrich-aware fatal log
func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	globalLogger.Fatal(ctx, msg, fields...)
}

// Instance methods для enrich loggers (logger)

func (l *logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(fieldsFromContext(ctx), fields...)
	l.zapLogger.Debug(msg, allFields...)
}

func (l *logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(fieldsFromContext(ctx), fields...)
	l.zapLogger.Info(msg, allFields...)
}

func (l *logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(fieldsFromContext(ctx), fields...)
	l.zapLogger.Warn(msg, allFields...)
}

func (l *logger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(fieldsFromContext(ctx), fields...)
	l.zapLogger.Error(msg, allFields...)
}

func (l *logger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append(fieldsFromContext(ctx), fields...)
	l.zapLogger.Fatal(msg, allFields...)
}

// parseLevel convert string level into zapcore.Level
func parseLevel(levelStr string) zapcore.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func addFieldFromContext[T any](
	fields *[]zap.Field,
	ctx context.Context,
	key contextx.CtxKey,
	extractor func(context.Context) (T, bool),
	fieldConstructor func(string, T) zap.Field,
) {
	if value, ok := extractor(ctx); ok {
		*fields = append(*fields, fieldConstructor(fmt.Sprint(key), value))
	}
}

func fieldsFromContext(ctx context.Context) []zap.Field {
	fields := make([]zap.Field, 0)

	addFieldFromContext(&fields, ctx, traceidctx.TraceIDKey, traceidctx.ExtractTraceId, zap.String)
	addFieldFromContext(&fields, ctx, traceidctx.TraceIDKey, traceidctx.ExtractTraceIDFromSpan, zap.String)
	addFieldFromContext(&fields, ctx, ipctx.IpKey, ipctx.ExtractIP, zap.String)
	addFieldFromContext(&fields, ctx, claimsctx.UserIDKey, claimsctx.ExtractUserID, zap.String)
	addFieldFromContext(&fields, ctx, claimsctx.UserEmailKey, claimsctx.ExtractUserEmail, zap.String)

	return fields
}

package logger

import (
	"context"
	"time"

	otelLog "go.opentelemetry.io/otel/log"
	"go.uber.org/zap/zapcore"
)

const emitTimeout = 500 * time.Millisecond

// SimpleOTLPCore convert zap-events into OpenTelemetry Records and emit to OTLP
type SimpleOTLPCore struct {
	otlpLogger otelLog.Logger
	level      zapcore.LevelEnabler
}

func NewSimpleOTLPCore(otlpLogger otelLog.Logger, level zapcore.LevelEnabler) *SimpleOTLPCore {
	return &SimpleOTLPCore{
		otlpLogger: otlpLogger,
		level:      level,
	}
}

func (c *SimpleOTLPCore) Enabled(level zapcore.Level) bool {
	return c.level.Enabled(level)
}

func (c *SimpleOTLPCore) With(_ []zapcore.Field) zapcore.Core {
	return &SimpleOTLPCore{
		otlpLogger: c.otlpLogger,
		level:      c.level,
	}
}

func (c *SimpleOTLPCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

func (c *SimpleOTLPCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	severity := mapZapToOtelSeverity(entry.Level)
	record := makeBaseRecord(entry, severity)
	if len(fields) > 0 {
		attrs := encodeFieldsToAttrs(fields)
		if len(attrs) > 0 {
			record.AddAttributes(attrs...)
		}
	}

	record.AddAttributes(
		otelLog.String("caller", entry.Caller.FullPath()),
	)

	c.emitWithTimeout(record)
	return nil
}

func (c *SimpleOTLPCore) Sync() error { return nil }

// mapZapToOtelSeverity — converter from zapcore.Level to OTLP severity
func mapZapToOtelSeverity(level zapcore.Level) otelLog.Severity {
	switch level {
	case zapcore.DebugLevel:
		return otelLog.SeverityDebug
	case zapcore.InfoLevel:
		return otelLog.SeverityInfo
	case zapcore.WarnLevel:
		return otelLog.SeverityWarn
	case zapcore.ErrorLevel:
		return otelLog.SeverityError
	default:
		return otelLog.SeverityInfo
	}
}

func makeBaseRecord(entry zapcore.Entry, sev otelLog.Severity) otelLog.Record {
	r := otelLog.Record{}
	r.SetSeverity(sev)
	r.SetBody(otelLog.StringValue(entry.Message))
	r.SetTimestamp(entry.Time)

	return r
}

func encodeFieldsToAttrs(fields []zapcore.Field) []otelLog.KeyValue {
	if len(fields) == 0 {
		return nil
	}

	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(enc)
	}

	attrs := make([]otelLog.KeyValue, 0, len(enc.Fields))
	for k, v := range enc.Fields {
		switch val := v.(type) {
		case string:
			attrs = append(attrs, otelLog.String(k, val))
		case bool:
			attrs = append(attrs, otelLog.Bool(k, val))
		case int64:
			attrs = append(attrs, otelLog.Int64(k, val))
		case float64:
			attrs = append(attrs, otelLog.Float64(k, val))
		case error:
			attrs = append(attrs, otelLog.String(k, val.Error()))
		}
	}

	return attrs
}

func (c *SimpleOTLPCore) emitWithTimeout(record otelLog.Record) {
	if c.otlpLogger == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), emitTimeout)
	defer cancel()
	c.otlpLogger.Emit(ctx, record)
}

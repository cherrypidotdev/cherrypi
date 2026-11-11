// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

type Config struct {
	ServiceName string

	ServiceVersion string
}

type Provider struct {
	traces  *trace.TracerProvider
	metrics *metric.MeterProvider
	logs    *sdklog.LoggerProvider

	restoreLogging func()
}

var enabled atomic.Bool

func Enabled() bool { return enabled.Load() }

func Setup(ctx context.Context, cfg Config) (*Provider, error) {
	res, err := newResource(ctx, cfg)
	if err != nil {

		log.Printf("OpenTelemetry resource is partial: %v", err)
	}

	p := &Provider{}
	restoreLogging := func() {}

	if signalEnabled("OTEL_TRACES_EXPORTER") {
		exporter, err := newTraceExporter(ctx)
		if err != nil {
			return nil, fmt.Errorf("create trace exporter: %w", err)
		}
		p.traces = trace.NewTracerProvider(
			trace.WithResource(res),
			trace.WithBatcher(exporter),
		)
	}

	if signalEnabled("OTEL_METRICS_EXPORTER") {
		exporter, err := newMetricExporter(ctx)
		if err != nil {
			_ = p.shutdown(ctx)

			return nil, fmt.Errorf("create metric exporter: %w", err)
		}
		p.metrics = metric.NewMeterProvider(
			metric.WithResource(res),
			metric.WithReader(metric.NewPeriodicReader(exporter)),
		)
	}

	if signalEnabled("OTEL_LOGS_EXPORTER") {
		exporter, err := newLogExporter(ctx)
		if err != nil {
			_ = p.shutdown(ctx)

			return nil, fmt.Errorf("create log exporter: %w", err)
		}
		p.logs = sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		)
		restoreLogging = bridgeStdLog(p.logs)
	}

	if p.traces != nil {
		otel.SetTracerProvider(p.traces)
	}
	if p.metrics != nil {
		otel.SetMeterProvider(p.metrics)
	}
	if p.logs != nil {
		otel.SetLoggerProvider(p.logs)
	}
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		_, _ = fmt.Fprintf(os.Stderr, "OpenTelemetry: %v\n", err)
	}))

	p.restoreLogging = restoreLogging
	enabled.Store(true)

	return p, nil
}

func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}

	enabled.Store(false)
	if p.restoreLogging != nil {
		p.restoreLogging()
		p.restoreLogging = nil
	}

	return p.shutdown(ctx)
}

func newResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	serviceName := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME"))
	if serviceName == "" {
		serviceName = cfg.ServiceName
	}
	if serviceName == "" {
		serviceName = "cherrypi-agency"
	}

	return resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithProcess(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
		),
	)
}

func (p *Provider) shutdown(ctx context.Context) error {
	var errs []error
	if p.logs != nil {
		errs = append(errs, p.logs.Shutdown(ctx))
		p.logs = nil
	}
	if p.traces != nil {
		errs = append(errs, p.traces.Shutdown(ctx))
		p.traces = nil
	}
	if p.metrics != nil {
		errs = append(errs, p.metrics.Shutdown(ctx))
		p.metrics = nil
	}

	return errors.Join(errs...)
}

func signalEnabled(envVar string) bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv(envVar)), "none")
}

func protocol(signal string) (string, error) {
	for _, key := range []string{
		"OTEL_EXPORTER_OTLP_" + signal + "_PROTOCOL",
		"OTEL_EXPORTER_OTLP_PROTOCOL",
	} {
		value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		switch value {
		case "":
			continue
		case "grpc":
			return "grpc", nil
		case "http", "http/protobuf":
			return "http/protobuf", nil
		default:
			return "", fmt.Errorf("%s=%q: want grpc or http/protobuf", key, value)
		}
	}

	return "grpc", nil
}

func newTraceExporter(ctx context.Context) (trace.SpanExporter, error) {
	proto, err := protocol("TRACES")
	if err != nil {
		return nil, err
	}
	if proto == "http/protobuf" {
		return otlptracehttp.New(ctx)
	}

	return otlptracegrpc.New(ctx)
}

func newMetricExporter(ctx context.Context) (metric.Exporter, error) {
	proto, err := protocol("METRICS")
	if err != nil {
		return nil, err
	}
	if proto == "http/protobuf" {
		return otlpmetrichttp.New(ctx)
	}

	return otlpmetricgrpc.New(ctx)
}

func newLogExporter(ctx context.Context) (sdklog.Exporter, error) {
	proto, err := protocol("LOGS")
	if err != nil {
		return nil, err
	}
	if proto == "http/protobuf" {
		return otlploghttp.New(ctx)
	}

	return otlploggrpc.New(ctx)
}

const shutdownTimeout = 10 * time.Second

func ShutdownContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), shutdownTimeout)
}

// LogStartup writes one line describing the effective pipeline. It is emitted
// after the providers are installed so the line itself travels over OTLP.
func (p *Provider) LogStartup() {
	if p == nil {
		return
	}

	if exportEndpoint("TRACES") == "default" {
		log.Print(
			"WARNING: no OTEL_EXPORTER_OTLP_ENDPOINT is set; telemetry goes to " +
				"the SDK default (localhost:4317 for grpc, localhost:4318 for " +
				"http/protobuf). Set the endpoint of your collector.",
		)
	}

	traces, metrics, logs := "off", "off", "off"
	if p.traces != nil {
		traces = "on"
	}
	if p.metrics != nil {
		metrics = "on"
	}
	if p.logs != nil {
		logs = "on"
	}

	proto, _ := protocol("TRACES")
	log.Printf(
		"OpenTelemetry enabled (traces=%s metrics=%s logs=%s protocol=%s endpoint=%s)",
		traces,
		metrics,
		logs,
		proto,
		exportEndpoint("TRACES"),
	)
}

func exportEndpoint(signal string) string {
	for _, key := range []string{
		"OTEL_EXPORTER_OTLP_" + signal + "_ENDPOINT",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
	} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}

	return "default"
}

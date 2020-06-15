package exporterhelper

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/obsreport"
)

// PushLogData is a helper function that is similar to ConsumeLogData but also returns
// the number of dropped logs.
type PushLogData func(ctx context.Context, td data.Logs) (droppedTimeSeries int, err error)

type logExporter struct {
	exporterFullName string
	pushLogData  		 PushLogData
	shutdown         Shutdown
}

func (me *logExporter) ConsumeLogs(ctx context.Context, md data.Logs) error {
	exporterCtx := obsreport.ExporterContext(ctx, me.exporterFullName)
	_, err := me.pushLogData(exporterCtx, md)
	return err
}

// NewMetricsExporter creates an MetricsExporter that can record metrics and can wrap every request with a Span.
// If no options are passed it just adds the exporter format as a tag in the Context.
// TODO: Add support for retries.
func NewLogExporter(config configmodels.Exporter, pushLogData PushLogData, options ...ExporterOption) (component.LogExporter, error) {
	if config == nil {
		return nil, errNilConfig
	}

	if pushLogData == nil {
		return nil, errNilPushMetricsData
	}

	opts := newExporterOptions(options...)

	// The default shutdown method always returns nil.
	if opts.shutdown == nil {
		opts.shutdown = func(context.Context) error { return nil }
	}

	return &logExporter{
		exporterFullName: config.Name(),
		pushLogData:  		pushLogData,
		shutdown:         opts.shutdown,
	}, nil
}

// Shutdown stops the exporter and is invoked during shutdown.
func (me *logExporter) Shutdown(ctx context.Context) error {
	return me.shutdown(ctx)
}

func (me *logExporter) Start(ctx context.Context, host component.Host) error {
	return nil
}
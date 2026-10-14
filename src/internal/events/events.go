package events

import (
	"context"
	"io"
	"time"

	"github.com/armon/go-metrics"

	"github.com/alcionai/corso/src/pkg/logger"
)

// keys for ease of use
const (
	// Event Keys
	RepoInit       = "Repo Init"
	RepoConnect    = "Repo Connect"
	BackupEnd      = "Backup End"
	RestoreEnd     = "Restore End"
	ExportEnd      = "Export End"
	MaintenanceEnd = "Maintenance End"

	// Event Data Keys
	BackupCreateTime = "backup_creation_time"
	BackupID         = "backup_id"
	DataRetrieved    = "data_retrieved"
	DataStored       = "data_stored"
	Duration         = "duration"
	EndTime          = "end_time"
	ItemsRead        = "items_read"
	ItemsWritten     = "items_written"
	Resources        = "resources"
	RestoreID        = "restore_id"
	ExportID         = "export_id"
	Service          = "service"
	StartTime        = "start_time"
	Status           = "status"

	// default values for keys
	RepoIDNotFound = "not_found"
)

type Eventer interface {
	Event(context.Context, string, map[string]any)
	Close() error
}

// Bus receives the events that operations report. Corso used to forward
// them to RudderStack as anonymous usage statistics; this fork removed that,
// so the bus drops every event. Operations keep reporting to it so a local
// sink can be added later without touching them again.
type Bus struct{}

func NewBus() Bus {
	return Bus{}
}

func (b Bus) Close() error {
	return nil
}

func (b Bus) Event(context.Context, string, map[string]any) {}

// ---------------------------------------------------------------------------
// metrics aggregation
// ---------------------------------------------------------------------------

type metricsCategory string

// metrics collection bucket
const (
	APICall metricsCategory = "api_call"
)

// configurations
const (
	reportInterval    = 1 * time.Minute
	retentionDuration = 2 * time.Minute
)

// NewMetrics embeds a metrics bus into the provided context.  The bus can be
// utilized with calls like Inc and Since.
func NewMetrics(ctx context.Context, w io.Writer) (context.Context, func()) {
	var (
		// report interval time-bounds metrics into buckets.  Retention
		// controls how long each interval sticks around.  Neither one controls
		// logging rates; that's handled by dumpMetrics().
		sink = metrics.NewInmemSink(reportInterval, retentionDuration)
		cfg  = metrics.DefaultConfig("corso")
		sig  = metrics.NewInmemSignal(sink, metrics.DefaultSignal, w)
	)

	cfg.EnableHostname = false
	cfg.EnableRuntimeMetrics = false

	gm, err := metrics.NewGlobal(cfg, sink)
	if err != nil {
		logger.CtxErr(ctx, err).Error("metrics bus constructor")
		sig.Stop()

		return ctx, func() {}
	}

	stop := make(chan struct{})
	go dumpMetrics(ctx, stop, sig)

	flush := func() {
		signalDump(ctx)
		time.Sleep(500 * time.Millisecond)
		close(stop)
		sig.Stop()
		gm.Shutdown()
	}

	// return context.WithValue(ctx, sinkCtxKey, sink), flush
	return ctx, flush
}

// dumpMetrics runs a loop that sends a os signal (SIGUSR1 on linux/mac, SIGBREAK on windows)
// every logging interval.  This syscall getts picked up by the metrics inmem signal and causes
// it to dump metrics to the provided writer (which should be the logger).
// Expectation is for users to call this in a goroutine.  Any signal or close() on the stop chan
// will exit the loop.
func dumpMetrics(ctx context.Context, stop <-chan struct{}, sig *metrics.InmemSignal) {
	tock := time.NewTicker(reportInterval)

	for {
		select {
		case <-tock.C:
			signalDump(ctx)
		case <-stop:
			return
		}
	}
}

// Inc increments the given category by 1.
func Inc(cat metricsCategory, keys ...string) {
	cats := append([]string{string(cat)}, keys...)
	metrics.IncrCounter(cats, 1)
}

// IncN increments the given category by N.
func IncN(n int, cat metricsCategory, keys ...string) {
	cats := append([]string{string(cat)}, keys...)
	metrics.IncrCounter(cats, float32(n))
}

// Since records the duration between the provided time and now, in millis.
func Since(start time.Time, cat metricsCategory, keys ...string) {
	cats := append([]string{string(cat)}, keys...)
	metrics.MeasureSince(cats, start)
}

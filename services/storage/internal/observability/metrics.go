package observability

import (
	"context"
	"fmt"
	"sync"
	"syscall"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/metric"
	otelmetric "go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
)

var (
	once         sync.Once
	initErr      error
	readBytes    otelmetric.Int64Counter
	writeBytes   otelmetric.Int64Counter
	diskUsage    otelmetric.Int64ObservableGauge
	registration otelmetric.Registration
)

func InitMetrics(dataDir string) error {
	once.Do(func() {
		readBytes, initErr = metric.NewInt64Counter(
			"storage_read_bytes_total",
			otelmetric.WithUnit("By"),
		)
		if initErr != nil {
			readBytes, _ = noopmetric.Meter{}.Int64Counter("storage_read_bytes_total")
			return
		}

		writeBytes, initErr = metric.NewInt64Counter(
			"storage_write_bytes_total",
			otelmetric.WithUnit("By"),
		)
		if initErr != nil {
			writeBytes, _ = noopmetric.Meter{}.Int64Counter("storage_write_bytes_total")
			return
		}

		diskUsage, initErr = metric.NewInt64ObservableGauge(
			"storage_disk_usage_bytes",
			otelmetric.WithUnit("By"),
		)
		if initErr != nil {
			diskUsage, _ = noopmetric.Meter{}.Int64ObservableGauge("storage_disk_usage_bytes")
			return
		}

		registration, initErr = metric.RegisterCallback(func(_ context.Context, observer otelmetric.Observer) error {
			usedBytes, err := getDiskUsageBytes(dataDir)
			if err != nil {
				return err
			}

			observer.ObserveInt64(diskUsage, usedBytes)
			return nil
		}, diskUsage)
	})

	return initErr
}

func AddReadBytes(ctx context.Context, bytes int64) {
	if bytes <= 0 || readBytes == nil {
		return
	}

	readBytes.Add(ctx, bytes)
}

func AddWriteBytes(ctx context.Context, bytes int64) {
	if bytes <= 0 || writeBytes == nil {
		return
	}

	writeBytes.Add(ctx, bytes)
}

func Shutdown() {
	if registration != nil {
		_ = registration.Unregister()
	}
}

func getDiskUsageBytes(dir string) (int64, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(dir, &stats); err != nil {
		return 0, fmt.Errorf("statfs data dir: %w", err)
	}

	usedBlocks := int64(stats.Blocks - stats.Bfree)
	blockSize := int64(stats.Bsize)

	return usedBlocks * blockSize, nil
}

package engine

import (
	"fmt"

	"github.com/ray/goreact/pkg/log"
)

// Close 优雅关闭引擎
func (r *reactor) Close() error {
	r.logger.Info("Shutting down engine")

	var errs []error

	// 关闭缓存
	if r.cache != nil {
		if err := r.cache.Close(); err != nil {
			r.logger.Error("Failed to close cache", log.Err(err))
			errs = append(errs, fmt.Errorf("cache close error: %w", err))
		}
	}

	// 关闭 metrics
	if r.metrics != nil {
		if err := r.metrics.Close(); err != nil {
			r.logger.Error("Failed to close metrics", log.Err(err))
			errs = append(errs, fmt.Errorf("metrics close error: %w", err))
		}
	}

	r.logger.Info("Engine shutdown complete")

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

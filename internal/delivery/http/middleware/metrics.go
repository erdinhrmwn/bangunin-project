package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests by method, path, and status.",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	asynqQueueSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "asynq_queue_size",
		Help: "Number of pending/active/scheduled/retry tasks per Asynq queue.",
	}, []string{"queue", "state"})
)

// Metrics records per-request Prometheus counters/histograms (FR-7.6).
func Metrics(c fiber.Ctx) error {
	start := time.Now()
	err := c.Next()

	status := strconv.Itoa(c.Response().StatusCode())
	httpRequestsTotal.WithLabelValues(c.Method(), c.Route().Path, status).Inc()
	httpRequestDuration.WithLabelValues(c.Method(), c.Route().Path).Observe(time.Since(start).Seconds())

	return err
}

// MetricsHandler serves the /metrics endpoint for Prometheus scraping.
func MetricsHandler() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.Handler())
}

// StartQueueSizeExporter polls Asynq queue depths every interval and updates
// the asynq_queue_size gauge. Runs until ctx is done.
func StartQueueSizeExporter(redisOpt asynq.RedisClientOpt, interval time.Duration) {
	inspector := asynq.NewInspector(redisOpt)
	go func() {
		for {
			queues, err := inspector.Queues()
			if err == nil {
				for _, q := range queues {
					if info, err := inspector.GetQueueInfo(q); err == nil {
						asynqQueueSize.WithLabelValues(q, "pending").Set(float64(info.Pending))
						asynqQueueSize.WithLabelValues(q, "active").Set(float64(info.Active))
						asynqQueueSize.WithLabelValues(q, "scheduled").Set(float64(info.Scheduled))
						asynqQueueSize.WithLabelValues(q, "retry").Set(float64(info.Retry))
					}
				}
			}
			time.Sleep(interval)
		}
	}()
}

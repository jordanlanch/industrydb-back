package metrics

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// Business metrics
	LeadsSearched    prometheus.Counter
	ExportsCreated   prometheus.Counter
	UsersRegistered  prometheus.Counter
	LoginAttempts    *prometheus.CounterVec
	SubscriptionsSold *prometheus.CounterVec

	// Database metrics
	DBQueryDuration *prometheus.HistogramVec
	DBConnections   prometheus.Gauge

	// Cache metrics
	CacheHits   *prometheus.CounterVec
	CacheMisses *prometheus.CounterVec
}

// New creates a new Metrics instance with all metrics registered
func New() *Metrics {
	m := &Metrics{
		// HTTP metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds",
				Buckets: prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: []float64{100, 1000, 5000, 10000, 50000, 100000, 500000, 1000000},
			},
			[]string{"method", "path"},
		),
		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: []float64{100, 1000, 5000, 10000, 50000, 100000, 500000, 1000000},
			},
			[]string{"method", "path"},
		),

		// Business metrics
		LeadsSearched: promauto.NewCounter(prometheus.CounterOpts{
			Name: "leads_searched_total",
			Help: "Total number of lead searches performed",
		}),
		ExportsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "exports_created_total",
			Help: "Total number of exports created",
		}),
		UsersRegistered: promauto.NewCounter(prometheus.CounterOpts{
			Name: "users_registered_total",
			Help: "Total number of users registered",
		}),
		LoginAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "login_attempts_total",
				Help: "Total number of login attempts",
			},
			[]string{"status"}, // success, failed
		),
		SubscriptionsSold: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "subscriptions_sold_total",
				Help: "Total number of subscriptions sold",
			},
			[]string{"tier"}, // starter, pro, business
		),

		// Database metrics
		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
			},
			[]string{"operation"}, // select, insert, update, delete
		),
		DBConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		}),

		// Cache metrics
		CacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type"}, // redis, memory
		),
		CacheMisses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type"},
		),
	}

	return m
}

// Middleware creates an Echo middleware for Prometheus metrics
func (m *Metrics) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			path := c.Path() // Use route pattern, not actual path (e.g., /api/v1/leads/:id)

			// Measure request size
			if req.ContentLength > 0 {
				m.HTTPRequestSize.WithLabelValues(req.Method, path).Observe(float64(req.ContentLength))
			}

			// Call next handler
			err := next(c)

			// Record metrics
			status := c.Response().Status
			duration := time.Since(start).Seconds()

			m.HTTPRequestsTotal.WithLabelValues(req.Method, path, strconv.Itoa(status)).Inc()
			m.HTTPRequestDuration.WithLabelValues(req.Method, path, strconv.Itoa(status)).Observe(duration)
			m.HTTPResponseSize.WithLabelValues(req.Method, path).Observe(float64(c.Response().Size))

			return err
		}
	}
}

// RecordLeadSearch increments leads searched counter
func (m *Metrics) RecordLeadSearch() {
	m.LeadsSearched.Inc()
}

// RecordExportCreated increments exports created counter
func (m *Metrics) RecordExportCreated() {
	m.ExportsCreated.Inc()
}

// RecordUserRegistered increments users registered counter
func (m *Metrics) RecordUserRegistered() {
	m.UsersRegistered.Inc()
}

// RecordLoginAttempt increments login attempts counter
func (m *Metrics) RecordLoginAttempt(success bool) {
	status := "failed"
	if success {
		status = "success"
	}
	m.LoginAttempts.WithLabelValues(status).Inc()
}

// RecordSubscriptionSold increments subscriptions sold counter
func (m *Metrics) RecordSubscriptionSold(tier string) {
	m.SubscriptionsSold.WithLabelValues(tier).Inc()
}

// RecordDBQuery records database query duration
func (m *Metrics) RecordDBQuery(operation string, duration time.Duration) {
	m.DBQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// UpdateDBConnections updates active database connections gauge
func (m *Metrics) UpdateDBConnections(count float64) {
	m.DBConnections.Set(count)
}

// RecordCacheHit increments cache hits counter
func (m *Metrics) RecordCacheHit(cacheType string) {
	m.CacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss increments cache misses counter
func (m *Metrics) RecordCacheMiss(cacheType string) {
	m.CacheMisses.WithLabelValues(cacheType).Inc()
}

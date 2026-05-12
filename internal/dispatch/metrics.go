package dispatch

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusMetrics struct {
	dispatchDuration       prometheus.Histogram
	driverAcceptTime       prometheus.Histogram
	dispatchRadiusAttempts *prometheus.CounterVec
	failedDispatches       prometheus.Counter
	dispatchTimeouts       prometheus.Counter
	activeSearches         prometheus.Gauge
	activeOrders           prometheus.Gauge
	staleDrivers           prometheus.Gauge
	wsConnections          prometheus.Gauge
	reconnects             prometheus.Counter
}

func NewPrometheusMetrics(registry *prometheus.Registry) *PrometheusMetrics {
	metrics := &PrometheusMetrics{
		dispatchDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "dispatch_duration_seconds",
			Help:    "Dispatch processing duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
		driverAcceptTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "driver_accept_time_seconds",
			Help:    "Time between offer creation and driver acceptance.",
			Buckets: prometheus.DefBuckets,
		}),
		dispatchRadiusAttempts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dispatch_radius_attempts_total",
			Help: "Dispatch attempts by radius.",
		}, []string{"radius_meters"}),
		failedDispatches: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "failed_dispatches_total",
			Help: "Total failed dispatches.",
		}),
		dispatchTimeouts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dispatch_timeouts_total",
			Help: "Total dispatch offer timeouts.",
		}),
		activeSearches: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "active_searches",
			Help: "Current number of searching orders discovered by recovery worker.",
		}),
		activeOrders: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "active_orders",
			Help: "Current active orders.",
		}),
		staleDrivers: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "stale_drivers",
			Help: "Current stale drivers detected by recovery workers.",
		}),
		wsConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "ws_connections",
			Help: "Current WebSocket connections.",
		}),
		reconnects: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "reconnects_total",
			Help: "Total realtime reconnects.",
		}),
	}

	registry.MustRegister(
		metrics.dispatchDuration,
		metrics.driverAcceptTime,
		metrics.dispatchRadiusAttempts,
		metrics.failedDispatches,
		metrics.dispatchTimeouts,
		metrics.activeSearches,
		metrics.activeOrders,
		metrics.staleDrivers,
		metrics.wsConnections,
		metrics.reconnects,
	)

	return metrics
}

func (metrics *PrometheusMetrics) ObserveDispatchDuration(duration time.Duration) {
	metrics.dispatchDuration.Observe(duration.Seconds())
}

func (metrics *PrometheusMetrics) ObserveDriverAcceptTime(duration time.Duration) {
	metrics.driverAcceptTime.Observe(duration.Seconds())
}

func (metrics *PrometheusMetrics) ObserveDispatchRadiusAttempt(radiusMeters int) {
	metrics.dispatchRadiusAttempts.WithLabelValues(formatRadius(radiusMeters)).Inc()
}

func (metrics *PrometheusMetrics) IncrementFailedDispatches() {
	metrics.failedDispatches.Inc()
}

func (metrics *PrometheusMetrics) IncrementDispatchTimeouts() {
	metrics.dispatchTimeouts.Inc()
}

func (metrics *PrometheusMetrics) SetActiveSearches(count int) {
	metrics.activeSearches.Set(float64(count))
}

func (metrics *PrometheusMetrics) SetActiveOrders(count int) {
	metrics.activeOrders.Set(float64(count))
}

func (metrics *PrometheusMetrics) SetStaleDrivers(count int) {
	metrics.staleDrivers.Set(float64(count))
}

func (metrics *PrometheusMetrics) SetWSConnections(count int) {
	metrics.wsConnections.Set(float64(count))
}

func (metrics *PrometheusMetrics) IncrementReconnects() {
	metrics.reconnects.Inc()
}

func formatRadius(radiusMeters int) string {
	return strconv.Itoa(radiusMeters)
}

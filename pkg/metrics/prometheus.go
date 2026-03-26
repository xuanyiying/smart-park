package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"service", "method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "path"},
	)

	VehicleEntryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vehicle_entry_total",
			Help: "Total number of vehicle entries",
		},
		[]string{"lot_id"},
	)

	VehicleExitTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vehicle_exit_total",
			Help: "Total number of vehicle exits",
		},
		[]string{"lot_id"},
	)

	PaymentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_total",
			Help: "Total number of payments",
		},
		[]string{"method", "status"},
	)

	PaymentAmount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_amount_total",
			Help: "Total payment amount",
		},
		[]string{"method"},
	)

	BillingCalculationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "billing_calculation_duration_seconds",
			Help:    "Duration of billing calculations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"lot_id"},
	)

	ActiveVehiclesGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_vehicles_current",
			Help: "Current number of active vehicles in parking lots",
		},
		[]string{"lot_id"},
	)

	CacheHitTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hit_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type"},
	)

	CacheMissTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_miss_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type"},
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Duration of database queries in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	NotificationSendTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_send_total",
			Help: "Total number of notifications sent",
		},
		[]string{"type", "status"},
	)

	MQTTCommandTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mqtt_command_total",
			Help: "Total number of MQTT commands sent",
		},
		[]string{"type", "status"},
	)
)

func RecordHTTPRequest(service, method, path string, status int, duration float64) {
	HTTPRequestsTotal.WithLabelValues(service, method, path, fmt.Sprintf("%d", status)).Inc()
	HTTPRequestDuration.WithLabelValues(service, method, path).Observe(duration)
}

func RecordVehicleEntry(lotID string) {
	VehicleEntryTotal.WithLabelValues(lotID).Inc()
}

func RecordVehicleExit(lotID string) {
	VehicleExitTotal.WithLabelValues(lotID).Inc()
}

func RecordPayment(method, status string, amount float64) {
	PaymentTotal.WithLabelValues(method, status).Inc()
	PaymentAmount.WithLabelValues(method).Add(amount)
}

func RecordBillingCalculation(lotID string, duration float64) {
	BillingCalculationDuration.WithLabelValues(lotID).Observe(duration)
}

func SetActiveVehicles(lotID string, count float64) {
	ActiveVehiclesGauge.WithLabelValues(lotID).Set(count)
}

func RecordCacheHit(cacheType string) {
	CacheHitTotal.WithLabelValues(cacheType).Inc()
}

func RecordCacheMiss(cacheType string) {
	CacheMissTotal.WithLabelValues(cacheType).Inc()
}

func RecordDatabaseQuery(operation, table string, duration float64) {
	DatabaseQueryDuration.WithLabelValues(operation, table).Observe(duration)
}

func RecordNotification(notificationType, status string) {
	NotificationSendTotal.WithLabelValues(notificationType, status).Inc()
}

func RecordMQTTCommand(commandType, status string) {
	MQTTCommandTotal.WithLabelValues(commandType, status).Inc()
}

package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "taxi_http_requests_total",
		Help: "Total HTTP requests handled by taxi API.",
	},
	[]string{"method", "path", "status"},
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
}

func PrometheusHTTPMetrics() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Next()

		path := context.FullPath()
		if path == "" {
			path = "unmatched"
		}
		httpRequestsTotal.WithLabelValues(
			context.Request.Method,
			path,
			strconv.Itoa(context.Writer.Status()),
		).Inc()
	}
}

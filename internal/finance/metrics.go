package finance

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	totalPlatformRevenue prometheus.Counter
	commissionRevenue    prometheus.Counter
	driverPayouts        prometheus.Counter
	parkRevenue          prometheus.Counter
	avgCommission        prometheus.Gauge
}

func NewMetrics(registry *prometheus.Registry) *Metrics {
	metrics := &Metrics{
		totalPlatformRevenue: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "total_platform_revenue_cents",
			Help: "Total completed orders revenue in cents.",
		}),
		commissionRevenue: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "commission_revenue_cents",
			Help: "Total platform commission revenue in cents.",
		}),
		driverPayouts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "driver_payouts_cents",
			Help: "Total driver payouts in cents.",
		}),
		parkRevenue: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "park_revenue_cents",
			Help: "Total taxi park revenue in cents.",
		}),
		avgCommission: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "avg_commission_per_order_cents",
			Help: "Average commission per completed order in cents.",
		}),
	}
	if registry != nil {
		registry.MustRegister(
			metrics.totalPlatformRevenue,
			metrics.commissionRevenue,
			metrics.driverPayouts,
			metrics.parkRevenue,
			metrics.avgCommission,
		)
	}
	return metrics
}

func (metrics *Metrics) ObserveSettlement(settlementCents int64, commissionCents int64, netCents int64, taxiParkOwned bool) {
	if metrics == nil {
		return
	}
	metrics.totalPlatformRevenue.Add(float64(settlementCents))
	metrics.commissionRevenue.Add(float64(commissionCents))
	if taxiParkOwned {
		metrics.parkRevenue.Add(float64(netCents))
	} else {
		metrics.driverPayouts.Add(float64(netCents))
	}
	metrics.avgCommission.Set(float64(commissionCents))
}

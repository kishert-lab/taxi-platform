package domain

import "testing"

func TestCalculateCommissionUsesIntegerCents(t *testing.T) {
	gross := Money{Amount: 100000, Currency: "RUB"}
	rate := CommissionRate{BasisPoints: 2000, Source: "taxi_park"}

	commission, net, err := CalculateCommission(gross, rate)
	if err != nil {
		t.Fatalf("calculate commission: %v", err)
	}

	if commission.Amount != 20000 {
		t.Fatalf("expected 20000 cents commission, got %d", commission.Amount)
	}
	if net.Amount != 80000 {
		t.Fatalf("expected 80000 cents net, got %d", net.Amount)
	}

	platformFee, err := CalculatePlatformFee(commission, CommissionRate{BasisPoints: 100, Source: "platform"})
	if err != nil {
		t.Fatalf("calculate platform fee: %v", err)
	}
	if platformFee.Amount != 200 {
		t.Fatalf("expected 200 cents platform fee, got %d", platformFee.Amount)
	}
}

func TestResolveCommissionRatePriority(t *testing.T) {
	city := int32(150)
	tariff := int32(200)
	taxiPark := int32(75)
	driver := int32(50)

	rate, err := ResolveCommissionRate(CommissionContext{
		PlatformDefaultBasisPoints: 100,
		CityBasisPoints:            &city,
		TariffBasisPoints:          &tariff,
		TaxiParkBasisPoints:        &taxiPark,
		DriverBasisPoints:          &driver,
	})
	if err != nil {
		t.Fatalf("resolve commission: %v", err)
	}

	if rate.BasisPoints != driver || rate.Source != "driver" {
		t.Fatalf("expected driver override, got %#v", rate)
	}
}

func TestCalculateCommissionZeroPercent(t *testing.T) {
	gross := Money{Amount: 100000, Currency: "RUB"}

	commission, net, err := CalculateCommission(gross, CommissionRate{BasisPoints: 0, Source: "taxi_park"})
	if err != nil {
		t.Fatalf("calculate commission: %v", err)
	}
	if commission.Amount != 0 {
		t.Fatalf("expected zero commission, got %d", commission.Amount)
	}
	if net.Amount != 100000 {
		t.Fatalf("expected full driver income, got %d", net.Amount)
	}
}

func TestCalculateCommissionHundredPercent(t *testing.T) {
	gross := Money{Amount: 100000, Currency: "RUB"}

	commission, net, err := CalculateCommission(gross, CommissionRate{BasisPoints: 10000, Source: "taxi_park"})
	if err != nil {
		t.Fatalf("calculate commission: %v", err)
	}
	if commission.Amount != 100000 {
		t.Fatalf("expected full commission, got %d", commission.Amount)
	}
	if net.Amount != 0 {
		t.Fatalf("expected zero driver income, got %d", net.Amount)
	}

	platformFee, err := CalculatePlatformFee(commission, CommissionRate{BasisPoints: 100, Source: "platform"})
	if err != nil {
		t.Fatalf("calculate platform fee: %v", err)
	}
	if platformFee.Amount != 1000 {
		t.Fatalf("expected 1000 cents platform fee, got %d", platformFee.Amount)
	}
}

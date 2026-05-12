package domain

import "testing"

func TestCalculateCommissionUsesIntegerCents(t *testing.T) {
	gross := Money{Amount: 50000, Currency: "RUB"}
	rate := CommissionRate{BasisPoints: 100, Source: "platform_default"}

	commission, net, err := CalculateCommission(gross, rate)
	if err != nil {
		t.Fatalf("calculate commission: %v", err)
	}

	if commission.Amount != 500 {
		t.Fatalf("expected 500 cents commission, got %d", commission.Amount)
	}
	if net.Amount != 49500 {
		t.Fatalf("expected 49500 cents net, got %d", net.Amount)
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

package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kishert-lab/taxi-platform/internal/auth"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/driverinvite"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/finance"
	"github.com/kishert-lab/taxi-platform/internal/security"
	"github.com/kishert-lab/taxi-platform/internal/taxipark"
	"go.uber.org/zap"
)

type inviteTestNotifier struct {
	mutex  sync.Mutex
	events []string
	err    error
}

func (notifier *inviteTestNotifier) Notify(_ context.Context, _ domain.DriverParkInvite, event string) error {
	notifier.mutex.Lock()
	defer notifier.mutex.Unlock()
	notifier.events = append(notifier.events, event)
	return notifier.err
}

type inviteFixture struct {
	pool       *pgxpool.Pool
	service    *driverinvite.Service
	repository *PostgresDriverParkInviteRepository
	driver     taxipark.CreateDriverResult
	owners     [3]uuid.UUID
	parks      [3]uuid.UUID
	city       uuid.UUID
	notifier   *inviteTestNotifier
}

func newInviteFixture(t *testing.T) *inviteFixture {
	t.Helper()
	dsn := os.Getenv("TAXI_INVITE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TAXI_INVITE_TEST_DATABASE_URL to a disposable database migrated through 000040")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	fixture := &inviteFixture{pool: pool, city: uuid.New(), notifier: &inviteTestNotifier{}}
	fixture.repository = NewPostgresDriverParkInviteRepository(pool)
	fixture.service = driverinvite.NewService(fixture.repository, fixture.notifier, 24*time.Hour, zap.NewNop())
	fixture.exec(t, `INSERT INTO cities(id,name,region,center) VALUES($1,$2,'test',ST_SetSRID(ST_MakePoint(56,58),4326))`, fixture.city, fixture.city.String())
	for index := range fixture.owners {
		fixture.owners[index] = uuid.New()
		fixture.parks[index] = uuid.New()
		fixture.exec(t, `INSERT INTO users(id,phone,role,registration_type) VALUES($1,$2,'taxi_park','taxi_park')`, fixture.owners[index], fixture.owners[index].String())
		fixture.exec(t, `INSERT INTO taxi_parks(id,owner_user_id,city_id,name,contact_phone,contact_email) VALUES($1,$2,$3,$4,'test','test@example.com')`, fixture.parks[index], fixture.owners[index], fixture.city, fmt.Sprintf("Park %d", index))
	}
	id := uuid.New()
	phone := fmt.Sprintf("+79%09d", uint64(id.ID())%1000000000)
	fixture.driver, err = NewPostgresTaxiParkSettingsRepository(pool).CreateDriverByOwnerUserID(context.Background(), fixture.owners[0], taxipark.CreateDriverRecord{Phone: phone, FirstName: "Driver", LastName: "Test", PasswordHash: "test", VerificationStatus: domain.ComplianceStatusDraft})
	if err != nil {
		t.Fatal(err)
	}
	return fixture
}
func (fixture *inviteFixture) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := fixture.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}
func (fixture *inviteFixture) invite(t *testing.T, park int) domain.DriverParkInvite {
	t.Helper()
	invite, err := fixture.service.Create(context.Background(), fixture.owners[park], fixture.driver.Phone)
	if err != nil {
		t.Fatal(err)
	}
	return invite
}
func (fixture *inviteFixture) assertPark(t *testing.T, park uuid.UUID) {
	t.Helper()
	var actual uuid.UUID
	if err := fixture.pool.QueryRow(context.Background(), `SELECT taxi_park_id FROM drivers WHERE id=$1`, fixture.driver.DriverID).Scan(&actual); err != nil {
		t.Fatal(err)
	}
	if actual != park {
		t.Fatalf("park=%s want %s", actual, park)
	}
}
func (fixture *inviteFixture) order(t *testing.T, status string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	passenger := uuid.New()
	fixture.exec(t, `INSERT INTO passengers(id,phone) VALUES($1,$2)`, passenger, passenger.String()[:30])
	fixture.exec(t, `INSERT INTO orders(id,passenger_id,driver_id,city_id,park_id,status,pickup_address,pickup_location,final_price) VALUES($1,$2,$3,$4,$5,$6,'test',ST_SetSRID(ST_MakePoint(56,58),4326),100)`, id, passenger, fixture.driver.DriverID, fixture.city, fixture.parks[0], status)
	return id
}

func TestDriverParkInvitesIntegration(t *testing.T) {
	ctx := context.Background()
	t.Run("registration unique phone and duplicate invitations", func(t *testing.T) {
		fixture := newInviteFixture(t)
		_, err := NewPostgresTaxiParkSettingsRepository(fixture.pool).CreateDriverByOwnerUserID(ctx, fixture.owners[1], taxipark.CreateDriverRecord{Phone: fixture.driver.Phone, VerificationStatus: domain.ComplianceStatusDraft})
		if !errors.Is(err, taxipark.ErrDriverPhoneAlreadyExists) {
			t.Fatalf("duplicate driver: %v", err)
		}
		first := fixture.invite(t, 1)
		second := fixture.invite(t, 1)
		if first.ID != second.ID {
			t.Fatal("pending invitation duplicated")
		}
		fixture.assertPark(t, fixture.parks[0])
		if len(fixture.notifier.events) != 1 {
			t.Fatalf("notifications=%v", fixture.notifier.events)
		}
		reconnected := driverinvite.NewService(fixture.repository, nil, time.Hour, zap.NewNop())
		invites, err := reconnected.List(ctx, fixture.driver.UserID, false)
		if err != nil || len(invites) != 1 {
			t.Fatalf("persistent notification: %v %v", invites, err)
		}
		if _, err := fixture.service.Create(ctx, fixture.owners[0], fixture.driver.Phone); !errors.Is(err, domain.ErrParkInviteSamePark) {
			t.Fatalf("same park: %v", err)
		}
	})
	t.Run("accept preserves account history money orders and token identity", func(t *testing.T) {
		fixture := newInviteFixture(t)
		order := fixture.order(t, "completed")
		fixture.exec(t, `UPDATE driver_balances SET available_balance_cents=35000 WHERE driver_id=$1`, fixture.driver.DriverID)
		fixture.exec(t, `INSERT INTO financial_transactions(driver_id,taxi_park_id,transaction_type,net_amount_cents) VALUES($1,$2,'manual_adjustment',35000)`, fixture.driver.DriverID, fixture.parks[0])
		first := fixture.invite(t, 1)
		other := fixture.invite(t, 2)
		manager := auth.NewTokenManager(auth.TokenManagerConfig{AccessSecret: "integration-access", RefreshSecret: "integration-refresh", Issuer: "test", AccessTTL: time.Hour, RefreshTTL: time.Hour})
		token, _, _, err := manager.IssueTokenPair(fixture.driver.UserID, domain.UserRoleDriver, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.service.Respond(ctx, fixture.driver.UserID, first.ID, domain.DriverParkInviteAccepted); err != nil {
			t.Fatal(err)
		}
		fixture.assertPark(t, fixture.parks[1])
		var closed, current int
		if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FILTER(WHERE left_at IS NOT NULL),count(*) FILTER(WHERE left_at IS NULL) FROM driver_park_history WHERE driver_id=$1`, fixture.driver.DriverID).Scan(&closed, &current); err != nil || closed != 1 || current != 1 {
			t.Fatalf("history=%d/%d err=%v", closed, current, err)
		}
		var acceptedAt *time.Time
		if err := fixture.pool.QueryRow(ctx, `SELECT accepted_at FROM driver_park_invites WHERE id=$1 AND status='accepted'`, first.ID).Scan(&acceptedAt); err != nil || acceptedAt == nil {
			t.Fatalf("accepted lifecycle was not persisted: %v %v", acceptedAt, err)
		}
		var auditEvents int
		if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM finance_audit_events WHERE payload->>'id'=$1 AND event_type IN ('driver_park_invite.accepted','driver.transferred_to_park')`, first.ID.String()).Scan(&auditEvents); err != nil || auditEvents != 2 {
			t.Fatalf("accept audit events=%d err=%v", auditEvents, err)
		}
		var otherStatus string
		if err := fixture.pool.QueryRow(ctx, `SELECT status FROM driver_park_invites WHERE id=$1`, other.ID).Scan(&otherStatus); err != nil || otherStatus != "cancelled" {
			t.Fatalf("other invite=%s err=%v", otherStatus, err)
		}
		claims, err := manager.ParseAccessToken(token)
		if err != nil {
			t.Fatal(err)
		}
		profile, err := NewPostgresDriverMobileRepository(fixture.pool).GetProfileByUserID(ctx, claims.Subject)
		if err != nil {
			t.Fatal(err)
		}
		if profile.TaxiParkID == nil || *profile.TaxiParkID != fixture.parks[1] {
			t.Fatal("old token resolves old park")
		}
		finances := NewPostgresFinanceRepository(fixture.pool)
		balance, err := finances.GetDriverBalance(ctx, fixture.driver.DriverID)
		if err != nil || balance.AvailableBalance.Amount != 0 {
			t.Fatalf("new balance=%v err=%v", balance, err)
		}
		oldBalance, err := finances.GetTaxiParkDriverBalance(ctx, fixture.owners[0], fixture.driver.DriverID)
		if err != nil || oldBalance.AvailableBalance.Amount != 35000 {
			t.Fatalf("old balance=%v err=%v", oldBalance, err)
		}
		snapshot, err := finances.GetOrderSnapshot(ctx, order)
		if err != nil || snapshot.TaxiParkID == nil || *snapshot.TaxiParkID != fixture.parks[0] {
			t.Fatalf("old order snapshot=%v err=%v", snapshot, err)
		}
		oldOrders, err := finances.ListTaxiParkOrders(ctx, fixture.owners[0], 100)
		if err != nil || len(oldOrders) != 1 {
			t.Fatalf("old orders=%v err=%v", oldOrders, err)
		}
		newOrders, err := finances.ListTaxiParkOrders(ctx, fixture.owners[1], 100)
		if err != nil || len(newOrders) != 0 {
			t.Fatalf("new park leaked orders=%v err=%v", newOrders, err)
		}
		var transactionPark uuid.UUID
		if err := fixture.pool.QueryRow(ctx, `SELECT taxi_park_id FROM financial_transactions WHERE driver_id=$1`, fixture.driver.DriverID).Scan(&transactionPark); err != nil || transactionPark != fixture.parks[0] {
			t.Fatalf("old finance changed: %v", err)
		}
		payout, err := finances.CreateDriverPayout(ctx, fixture.owners[0], fixture.driver.DriverID, finance.CreateDriverPayoutInput{AmountCents: 1000})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := finances.MarkDriverPayoutPaid(ctx, fixture.owners[0], payout.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := finances.MarkDriverPayoutPaid(ctx, fixture.owners[0], payout.ID); err == nil {
			t.Fatal("payout paid twice")
		}
		oldBalance, err = finances.GetTaxiParkDriverBalance(ctx, fixture.owners[0], fixture.driver.DriverID)
		if err != nil || oldBalance.AvailableBalance.Amount != 34000 {
			t.Fatalf("old payout=%v err=%v", oldBalance, err)
		}
	})
	t.Run("park list hides source park and changed context blocks transfer", func(t *testing.T) {
		fixture := newInviteFixture(t)
		invite := fixture.invite(t, 1)
		parkInvites, err := fixture.service.List(ctx, fixture.owners[1], true)
		if err != nil || len(parkInvites) != 1 {
			t.Fatalf("park invites: %v %v", parkInvites, err)
		}
		if parkInvites[0].FromTaxiParkID != nil || parkInvites[0].FromTaxiParkName != "" {
			t.Fatalf("source park leaked to destination: %+v", parkInvites[0])
		}
		driverInvites, err := fixture.service.List(ctx, fixture.driver.UserID, false)
		if err != nil || len(driverInvites) != 1 || driverInvites[0].FromTaxiParkID == nil || *driverInvites[0].FromTaxiParkID != fixture.parks[0] {
			t.Fatalf("driver source context missing: %v %v", driverInvites, err)
		}
		fixture.exec(t, `UPDATE drivers SET taxi_park_id=$2 WHERE id=$1`, fixture.driver.DriverID, fixture.parks[2])
		if _, err := fixture.service.Respond(ctx, fixture.driver.UserID, invite.ID, domain.DriverParkInviteAccepted); !errors.Is(err, domain.ErrParkInviteContextChanged) {
			t.Fatalf("changed context accept=%v", err)
		}
		fixture.assertPark(t, fixture.parks[2])
	})
	t.Run("notification failure does not roll back committed invitation", func(t *testing.T) {
		fixture := newInviteFixture(t)
		fixture.notifier.err = errors.New("notification transport unavailable")
		invite, err := fixture.service.Create(ctx, fixture.owners[1], fixture.driver.Phone)
		if err != nil {
			t.Fatal(err)
		}
		invitations, err := fixture.service.List(ctx, fixture.driver.UserID, false)
		if err != nil || len(invitations) != 1 || invitations[0].ID != invite.ID || invitations[0].Status != domain.DriverParkInvitePending {
			t.Fatalf("committed invitation: %v %v", invitations, err)
		}
		fixture.assertPark(t, fixture.parks[0])
	})
	t.Run("decline cancel expired and foreign invitations", func(t *testing.T) {
		fixture := newInviteFixture(t)
		invite := fixture.invite(t, 1)
		if _, err := fixture.service.Respond(ctx, uuid.New(), invite.ID, domain.DriverParkInviteAccepted); !errors.Is(err, domain.ErrParkInviteNotFound) {
			t.Fatalf("foreign accept=%v", err)
		}
		if _, err := fixture.service.Respond(ctx, fixture.owners[2], invite.ID, domain.DriverParkInviteCancelled); !errors.Is(err, domain.ErrParkInviteNotFound) {
			t.Fatalf("foreign cancel=%v", err)
		}
		if _, err := fixture.service.Respond(ctx, fixture.driver.UserID, invite.ID, domain.DriverParkInviteDeclined); err != nil {
			t.Fatal(err)
		}
		fixture.assertPark(t, fixture.parks[0])
		invite = fixture.invite(t, 1)
		if _, err := fixture.service.Respond(ctx, fixture.owners[1], invite.ID, domain.DriverParkInviteCancelled); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.service.Respond(ctx, fixture.driver.UserID, invite.ID, domain.DriverParkInviteAccepted); !errors.Is(err, domain.ErrParkInviteClosed) {
			t.Fatalf("cancelled accept=%v", err)
		}
		invite = fixture.invite(t, 1)
		fixture.exec(t, `UPDATE driver_park_invites SET created_at=now()-interval '2 days',expires_at=now()-interval '1 day' WHERE id=$1`, invite.ID)
		if _, err := fixture.service.Respond(ctx, fixture.driver.UserID, invite.ID, domain.DriverParkInviteAccepted); !errors.Is(err, domain.ErrParkInviteExpired) {
			t.Fatalf("expired accept=%v", err)
		}
		if replacement := fixture.invite(t, 1); replacement.ID == invite.ID {
			t.Fatal("expired invite reused")
		}
	})
	t.Run("active order and online driver block transfer", func(t *testing.T) {
		fixture := newInviteFixture(t)
		invite := fixture.invite(t, 1)
		order := fixture.order(t, "in_progress")
		if _, err := fixture.service.Respond(ctx, fixture.driver.UserID, invite.ID, domain.DriverParkInviteAccepted); !errors.Is(err, domain.ErrParkInviteDriverWorking) {
			t.Fatalf("active order=%v", err)
		}
		fixture.exec(t, `UPDATE orders SET status='completed' WHERE id=$1`, order)
		fixture.exec(t, `UPDATE drivers SET status='online' WHERE id=$1`, fixture.driver.DriverID)
		if _, err := fixture.service.Respond(ctx, fixture.driver.UserID, invite.ID, domain.DriverParkInviteAccepted); !errors.Is(err, domain.ErrParkInviteDriverWorking) {
			t.Fatalf("online=%v", err)
		}
		fixture.assertPark(t, fixture.parks[0])
	})
	t.Run("concurrent accept has exactly one winner", func(t *testing.T) {
		fixture := newInviteFixture(t)
		invites := []domain.DriverParkInvite{fixture.invite(t, 1), fixture.invite(t, 2)}
		start := make(chan struct{})
		results := make(chan error, 2)
		for _, invite := range invites {
			go func(id uuid.UUID) {
				<-start
				_, err := fixture.service.Respond(ctx, fixture.driver.UserID, id, domain.DriverParkInviteAccepted)
				results <- err
			}(invite.ID)
		}
		close(start)
		success := 0
		for range invites {
			err := <-results
			if err == nil {
				success++
			} else if !errors.Is(err, domain.ErrParkInviteClosed) {
				t.Fatal(err)
			}
		}
		if success != 1 {
			t.Fatalf("winners=%d", success)
		}
	})
	t.Run("concurrent create returns one invite", func(t *testing.T) {
		fixture := newInviteFixture(t)
		results := make(chan domain.DriverParkInvite, 2)
		failures := make(chan error, 2)
		for range 2 {
			go func() {
				result, err := fixture.service.Create(ctx, fixture.owners[1], fixture.driver.Phone)
				results <- result
				failures <- err
			}()
		}
		first, second := <-results, <-results
		for range 2 {
			if err := <-failures; err != nil {
				t.Fatal(err)
			}
		}
		if first.ID != second.ID {
			t.Fatal("duplicate concurrent invites")
		}
	})
	t.Run("unaffiliated driver and inactive destination", func(t *testing.T) {
		fixture := newInviteFixture(t)
		fixture.exec(t, `UPDATE drivers SET taxi_park_id=NULL WHERE id=$1`, fixture.driver.DriverID)
		invite := fixture.invite(t, 1)
		if invite.FromTaxiParkID != nil {
			t.Fatal("unexpected source park")
		}
		fixture.exec(t, `UPDATE taxi_parks SET deleted_at=now() WHERE id=$1`, fixture.parks[1])
		if _, err := fixture.service.Respond(ctx, fixture.driver.UserID, invite.ID, domain.DriverParkInviteAccepted); !errors.Is(err, domain.ErrParkInviteInactivePark) {
			t.Fatalf("inactive=%v", err)
		}
		fixture.exec(t, `UPDATE taxi_parks SET deleted_at=NULL WHERE id=$1`, fixture.parks[1])
		if _, err := fixture.service.Respond(ctx, fixture.driver.UserID, invite.ID, domain.DriverParkInviteAccepted); err != nil {
			t.Fatal(err)
		}
		fixture.assertPark(t, fixture.parks[1])
	})
	t.Run("transaction rolls back when audit fails", func(t *testing.T) {
		fixture := newInviteFixture(t)
		invite := fixture.invite(t, 1)
		broken := driverinvite.NewService(failingInviteRepository{fixture.repository}, nil, time.Hour, zap.NewNop())
		if _, err := broken.Respond(ctx, fixture.driver.UserID, invite.ID, domain.DriverParkInviteAccepted); err == nil {
			t.Fatal("expected audit failure")
		}
		fixture.assertPark(t, fixture.parks[0])
		invites, err := fixture.service.List(ctx, fixture.driver.UserID, false)
		if err != nil || len(invites) != 1 || invites[0].Status != domain.DriverParkInvitePending {
			t.Fatalf("partial transfer: %v %v", invites, err)
		}
	})
	t.Run("late settlement stays with old park and return restores its balance", func(t *testing.T) {
		fixture := newInviteFixture(t)
		order := fixture.order(t, "completed")
		invitation := fixture.invite(t, 1)
		if _, err := fixture.service.Respond(ctx, fixture.driver.UserID, invitation.ID, domain.DriverParkInviteAccepted); err != nil {
			t.Fatal(err)
		}
		finances := NewPostgresFinanceRepository(fixture.pool)
		settlement, err := finance.NewService(finances, zap.NewNop()).SettleCompletedOrder(ctx, order)
		if err != nil {
			t.Fatal(err)
		}
		if settlement.TaxiParkID == nil || *settlement.TaxiParkID != fixture.parks[0] {
			t.Fatal("late settlement moved to new park")
		}
		balance, err := finances.GetDriverBalance(ctx, fixture.driver.DriverID)
		if err != nil || balance.AvailableBalance.Amount != 0 {
			t.Fatalf("late settlement leaked: %v %v", balance, err)
		}
		invitation = fixture.invite(t, 0)
		if _, err := fixture.service.Respond(ctx, fixture.driver.UserID, invitation.ID, domain.DriverParkInviteAccepted); err != nil {
			t.Fatal(err)
		}
		balance, err = finances.GetDriverBalance(ctx, fixture.driver.DriverID)
		if err != nil || balance.AvailableBalance != settlement.NetAmount {
			t.Fatalf("restored balance: %v %v", balance, err)
		}
	})
	t.Run("login without park and driver push registration", func(t *testing.T) {
		fixture := newInviteFixture(t)
		hasher := security.NewBCryptPasswordHasher(4)
		hash, err := hasher.HashPassword("integration-password")
		if err != nil {
			t.Fatal(err)
		}
		fixture.exec(t, `UPDATE drivers SET taxi_park_id=NULL WHERE id=$1`, fixture.driver.DriverID)
		fixture.exec(t, `UPDATE users SET password_hash=$2 WHERE id=$1`, fixture.driver.UserID, hash)
		manager := auth.NewTokenManager(auth.TokenManagerConfig{AccessSecret: "integration-access", RefreshSecret: "integration-refresh", Issuer: "test", AccessTTL: time.Hour, RefreshTTL: 24 * time.Hour})
		login := auth.NewMobileService(auth.NewMobileServiceParams{UserRepository: NewPostgresUserRepository(fixture.pool), RefreshTokenRepository: NewPostgresRefreshTokenRepository(fixture.pool), PasswordHasher: hasher, TokenManager: manager, Logger: zap.NewNop()})
		tokens, err := login.StartLogin(ctx, dto.AuthLoginRequest{Phone: fixture.driver.Phone, Password: "integration-password", Role: domain.UserRoleDriver})
		if err != nil {
			t.Fatal(err)
		}
		if tokens.AccessToken == "" {
			t.Fatal("no access token without park")
		}
		if err := fixture.service.SavePushToken(ctx, fixture.driver.UserID, "test-"+fixture.driver.UserID.String(), "android"); err != nil {
			t.Fatal(err)
		}
		pushTokens, err := fixture.repository.ListDriverPushTokens(ctx, fixture.driver.DriverID)
		if err != nil || len(pushTokens) != 1 {
			t.Fatalf("push tokens: %v %v", pushTokens, err)
		}
		if err := fixture.service.DeletePushToken(ctx, fixture.driver.UserID, pushTokens[0]); err != nil {
			t.Fatal(err)
		}
		pushTokens, err = fixture.repository.ListDriverPushTokens(ctx, fixture.driver.DriverID)
		if err != nil || len(pushTokens) != 0 {
			t.Fatalf("push logout: %v %v", pushTokens, err)
		}
	})
}

type failingInviteRepository struct {
	*PostgresDriverParkInviteRepository
}

func (repository failingInviteRepository) Transact(ctx context.Context, callback func(driverinvite.Transaction) error) error {
	return repository.PostgresDriverParkInviteRepository.Transact(ctx, func(transaction driverinvite.Transaction) error {
		return callback(failingInviteTransaction{transaction})
	})
}

type failingInviteTransaction struct{ driverinvite.Transaction }

func (transaction failingInviteTransaction) Audit(context.Context, uuid.UUID, string, domain.DriverParkInvite) error {
	return errors.New("injected audit failure")
}

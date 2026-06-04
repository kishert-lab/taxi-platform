package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/configs"
	adminapp "github.com/kishert-lab/taxi-platform/internal/admin"
	"github.com/kishert-lab/taxi-platform/internal/database"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/repository"
	"github.com/kishert-lab/taxi-platform/internal/security"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "create-taxi-park":
		err = runCreateTaxiPark(ctx, os.Args[2:])
	case "list-taxi-parks":
		err = runListTaxiParks(ctx, os.Args[2:])
	case "reset-password":
		err = runResetPassword(ctx, os.Args[2:])
	case "cities", "list-cities":
		err = runListCities(ctx, os.Args[2:])
	case "monitor":
		err = runMonitor(ctx, os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
		return
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runListTaxiParks(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("list-taxi-parks", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	var command adminapp.ListTaxiParkAccountsCommand
	var output string
	flags.IntVar(&command.Limit, "limit", 100, "maximum number of taxi park accounts to list")
	flags.StringVar(&command.Search, "search", "", "optional search by park name, legal name, tax id, phone, or email")
	flags.BoolVar(&command.IncludeDeleted, "include-deleted", false, "include soft-deleted taxi parks")
	flags.StringVar(&output, "output", "text", "output format: text or json")
	if err := flags.Parse(args); err != nil {
		return err
	}

	service, closeService, err := newAdminService(ctx)
	if err != nil {
		return err
	}
	defer closeService()

	accounts, err := service.ListTaxiParkAccounts(ctx, command)
	if err != nil {
		return err
	}

	return printTaxiParkAccounts(accounts, output)
}

func runCreateTaxiPark(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("create-taxi-park", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	var command adminapp.CreateTaxiParkCommand
	var cityID string
	var output string
	flags.StringVar(&command.Phone, "phone", "", "owner phone in E.164 format, for example +79990000000")
	flags.StringVar(&command.Email, "email", "", "owner email")
	flags.StringVar(&command.Password, "password", "", "owner password; generated when empty")
	flags.StringVar(&command.FirstName, "first-name", "", "owner first name")
	flags.StringVar(&command.LastName, "last-name", "", "owner last name")
	flags.StringVar(&cityID, "city-id", "", "city UUID")
	flags.StringVar(&command.Name, "name", "", "taxi park display name")
	flags.StringVar(&command.LegalName, "legal-name", "", "taxi park legal name")
	flags.StringVar(&command.TaxID, "tax-id", "", "tax identifier, INN by default for Russia")
	flags.StringVar(&command.CommissionPercent, "commission-percent", "", "optional commission percent override, for example 1.50")
	flags.BoolVar(&command.Verified, "verified", false, "mark taxi park as verified")
	flags.BoolVar(&command.AcceptDocuments, "accept-documents", false, "required explicit confirmation of personal data and terms documents")
	flags.StringVar(&command.PrivacyPolicyVersion, "privacy-policy-version", "1.0", "accepted privacy policy version")
	flags.StringVar(&command.TermsVersion, "terms-version", "1.0", "accepted terms of service version")
	flags.StringVar(&command.ConsentIP, "consent-ip", "console", "audit IP/source for document acceptance")
	flags.StringVar(&command.ConsentUserAgent, "consent-user-agent", "taxi-admin-cli", "audit user agent/source for document acceptance")
	flags.StringVar(&output, "output", "text", "output format: text or json")
	if err := flags.Parse(args); err != nil {
		return err
	}

	parsedCityID, err := uuid.Parse(strings.TrimSpace(cityID))
	if err != nil {
		return fmt.Errorf("parse city-id: %w", err)
	}
	command.CityID = parsedCityID

	service, closeService, err := newAdminService(ctx)
	if err != nil {
		return err
	}
	defer closeService()

	result, err := service.CreateTaxiPark(ctx, command)
	if err != nil {
		return err
	}

	return printCreateTaxiParkResult(result, output)
}

func runResetPassword(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("reset-password", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	var command adminapp.ResetPasswordCommand
	var role string
	var output string
	flags.StringVar(&command.Phone, "phone", "", "user phone in E.164 format, for example +79990000000")
	flags.StringVar(&role, "role", string(domain.UserRoleTaxiPark), "user role: passenger, driver, taxi_park, admin, dispatcher")
	flags.StringVar(&command.Password, "password", "", "new password; generated when empty")
	flags.StringVar(&output, "output", "text", "output format: text or json")
	if err := flags.Parse(args); err != nil {
		return err
	}
	command.Role = domain.UserRole(strings.TrimSpace(role))

	service, closeService, err := newAdminService(ctx)
	if err != nil {
		return err
	}
	defer closeService()

	result, err := service.ResetPassword(ctx, command)
	if err != nil {
		return err
	}

	return printResetPasswordResult(result, output)
}

func runListCities(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("cities", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	var output string
	flags.StringVar(&output, "output", "text", "output format: text or json")
	if err := flags.Parse(args); err != nil {
		return err
	}

	service, closeService, err := newAdminService(ctx)
	if err != nil {
		return err
	}
	defer closeService()

	cities, err := service.ListCities(ctx)
	if err != nil {
		return err
	}

	return printCities(cities, output)
}

func newAdminService(ctx context.Context) (*adminapp.Service, func(), error) {
	config, err := configs.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}

	postgresPool, err := database.NewPostgres(ctx, config.Database)
	if err != nil {
		return nil, nil, fmt.Errorf("connect postgres: %w", err)
	}

	adminRepository := repository.NewPostgresAdminRepository(postgresPool)
	passwordHasher := security.NewBCryptPasswordHasher(config.Security.BCryptCost)
	return adminapp.NewService(adminRepository, passwordHasher), postgresPool.Close, nil
}

func printTaxiParkAccounts(accounts []adminapp.TaxiParkAccount, output string) error {
	if output == "json" {
		return printJSON(struct {
			Accounts []adminapp.TaxiParkAccount `json:"accounts"`
			Count    int                        `json:"count"`
		}{Accounts: accounts, Count: len(accounts)})
	}
	if output != "text" {
		return fmt.Errorf("unsupported output format %q", output)
	}

	if len(accounts) == 0 {
		fmt.Println("taxi park accounts not found")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "TAXI_PARK_ID\tNAME\tCITY\tOWNER_PHONE\tOWNER_EMAIL\tSTATUS\tCOMMISSION\tBALANCE_RUB\tOWNER_ACTIVE\tCREATED_AT")
	for _, account := range accounts {
		commissionPercent := account.CommissionPercent
		if commissionPercent == "" {
			commissionPercent = "-"
		}
		ownerEmail := account.OwnerEmail
		if ownerEmail == "" {
			ownerEmail = "-"
		}
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%t\t%s\n",
			account.TaxiParkID,
			account.Name,
			account.CityName,
			account.OwnerPhone,
			ownerEmail,
			account.VerificationStatus,
			commissionPercent,
			formatCents(account.BalanceCents),
			account.IsOwnerActive,
			account.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush taxi park accounts table: %w", err)
	}

	return nil
}

func printCreateTaxiParkResult(result adminapp.CreateTaxiParkResult, output string) error {
	if output == "json" {
		return printJSON(result)
	}
	if output != "text" {
		return fmt.Errorf("unsupported output format %q", output)
	}

	fmt.Println("taxi park created")
	fmt.Printf("user_id: %s\n", result.UserID)
	fmt.Printf("taxi_park_id: %s\n", result.TaxiParkID)
	fmt.Printf("phone: %s\n", result.Phone)
	fmt.Printf("email: %s\n", result.Email)
	if result.PasswordGenerated {
		fmt.Printf("generated_password: %s\n", result.GeneratedPassword)
	}

	return nil
}

func printResetPasswordResult(result adminapp.ResetPasswordCommandResult, output string) error {
	if output == "json" {
		return printJSON(result)
	}
	if output != "text" {
		return fmt.Errorf("unsupported output format %q", output)
	}

	fmt.Println("password reset")
	fmt.Printf("user_id: %s\n", result.UserID)
	fmt.Printf("phone: %s\n", result.Phone)
	fmt.Printf("role: %s\n", result.Role)
	fmt.Printf("revoked_refresh_tokens: %d\n", result.RevokedTokenCount)
	if result.PasswordGenerated {
		fmt.Printf("generated_password: %s\n", result.GeneratedPassword)
	}

	return nil
}

func printCities(cities []adminapp.CityRecord, output string) error {
	if output == "json" {
		return printJSON(struct {
			Cities []adminapp.CityRecord `json:"cities"`
			Count  int                   `json:"count"`
		}{Cities: cities, Count: len(cities)})
	}
	if output != "text" {
		return fmt.Errorf("unsupported output format %q", output)
	}

	if len(cities) == 0 {
		fmt.Println("cities not found")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tNAME\tREGION\tCOUNTRY\tTIMEZONE\tLATITUDE\tLONGITUDE\tACTIVE")
	for _, city := range cities {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%.6f\t%.6f\t%t\n",
			city.ID,
			city.Name,
			city.Region,
			city.CountryCode,
			city.Timezone,
			city.Latitude,
			city.Longitude,
			city.IsActive,
		)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush cities output: %w", err)
	}

	return nil
}

func formatCents(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	return fmt.Sprintf("%s%d.%02d", sign, value/100, value%100)
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode json output: %w", err)
	}

	return nil
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `usage:
  go run ./cmd/admin create-taxi-park --phone +79990000000 --email park@example.com --city-id <uuid> --name "City Taxi" --accept-documents
  go run ./cmd/admin list-taxi-parks --limit 100
  go run ./cmd/admin reset-password --phone +79990000000 --role taxi_park
  go run ./cmd/admin cities
  go run ./cmd/admin monitor
  go run ./cmd/admin monitor --inline

commands:
  create-taxi-park  create taxi park owner user, taxi park profile, settings and consent audit
  list-taxi-parks   list taxi park accounts with owner contacts and finance status
  reset-password    reset user password by phone and role, revoking active refresh tokens
  cities            list cities with UUIDs for taxi park creation
  monitor           open live console monitoring dashboard`)
}

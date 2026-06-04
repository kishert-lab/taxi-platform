package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/kishert-lab/taxi-platform/configs"
	adminapp "github.com/kishert-lab/taxi-platform/internal/admin"
	"github.com/kishert-lab/taxi-platform/internal/database"
	redisclient "github.com/kishert-lab/taxi-platform/internal/redis"
	"github.com/kishert-lab/taxi-platform/internal/repository"
	"github.com/kishert-lab/taxi-platform/internal/security"
)

const (
	defaultMonitorWidth  = 120
	defaultMonitorHeight = 36
	defaultMonitorEvery  = 2 * time.Second
)

var requestMetricPattern = regexp.MustCompile(`^taxi_http_requests_total(?:\{[^}]*\})?\s+([0-9.eE+-]+)`)

type monitorOptions struct {
	interval   time.Duration
	metricsURL string
	width      int
	height     int
	once       bool
	inline     bool
}

type monitorResources struct {
	config      *configs.Config
	service     *adminapp.Service
	redisClient *goredis.Client
	close       func()
}

type monitorSnapshot struct {
	database              adminapp.MonitorDatabaseSnapshot
	databaseError         error
	redisOnlineDrivers    int64
	redisConnected        bool
	redisError            error
	httpRequestsTotal     float64
	httpRequestsPerMinute *float64
	metricsConnected      bool
	metricsError          error
	collectedAt           time.Time
}

func runMonitor(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("monitor", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	options := monitorOptions{}
	flags.DurationVar(&options.interval, "interval", defaultMonitorEvery, "refresh interval")
	flags.StringVar(&options.metricsURL, "metrics-url", "", "Prometheus metrics URL; defaults to local API /metrics")
	flags.IntVar(&options.width, "width", defaultMonitorWidth, "dashboard width in terminal characters")
	flags.IntVar(&options.height, "height", defaultMonitorHeight, "dashboard height in terminal rows")
	flags.BoolVar(&options.once, "once", false, "render one snapshot and exit")
	flags.BoolVar(&options.inline, "inline", false, "render in current terminal instead of opening a system terminal window")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if options.interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}
	if options.width < 80 {
		return fmt.Errorf("width must be at least 80")
	}
	if options.height < 24 {
		return fmt.Errorf("height must be at least 24")
	}
	if shouldLaunchMonitorWindow(options) {
		return launchMonitorWindow(options, args)
	}

	resources, err := newMonitorResources(ctx)
	if err != nil {
		return err
	}
	defer resources.close()

	if strings.TrimSpace(options.metricsURL) == "" {
		options.metricsURL = fmt.Sprintf("http://127.0.0.1:%d/metrics", resources.config.Server.Port)
	}

	var previousRequests *metricSample
	if !options.once {
		enterMonitorScreen(os.Stdout)
		defer leaveMonitorScreen(os.Stdout)
	}

	for {
		snapshot := collectMonitorSnapshot(ctx, resources, options.metricsURL, previousRequests)
		if snapshot.metricsConnected {
			previousRequests = &metricSample{
				value:       snapshot.httpRequestsTotal,
				collectedAt: snapshot.collectedAt,
			}
		}

		renderMonitor(os.Stdout, snapshot, options)
		if options.once {
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(options.interval):
		}
	}
}

func shouldLaunchMonitorWindow(options monitorOptions) bool {
	if runtime.GOOS != "windows" || options.once || options.inline {
		return false
	}
	return os.Getenv("TAXI_ADMIN_MONITOR_INLINE") == ""
}

func launchMonitorWindow(options monitorOptions, args []string) error {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	childArguments := append([]string{"monitor", "--inline"}, args...)
	commandLine := fmt.Sprintf(
		"$ErrorActionPreference='Stop'; "+
			"$env:TAXI_ADMIN_MONITOR_INLINE='1'; "+
			"$host.UI.RawUI.WindowTitle='Taxi Platform Monitor'; "+
			"try { mode con: cols=%d lines=%d | Out-Null } catch {}; "+
			"Set-Location -LiteralPath %s; "+
			"go run ./cmd/admin %s",
		options.width,
		options.height,
		powerShellSingleQuoted(workingDirectory),
		joinPowerShellArguments(childArguments),
	)

	command := exec.Command("cmd", "/C", "start", "Taxi Platform Monitor", "powershell.exe", "-NoExit", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", commandLine)
	if err := command.Start(); err != nil {
		return fmt.Errorf("launch monitor terminal window: %w", err)
	}

	fmt.Fprintln(os.Stdout, "monitor opened in a system terminal window")
	return nil
}

func joinPowerShellArguments(arguments []string) string {
	quoted := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		quoted = append(quoted, powerShellSingleQuoted(argument))
	}
	return strings.Join(quoted, " ")
}

func powerShellSingleQuoted(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func newMonitorResources(ctx context.Context) (monitorResources, error) {
	config, err := configs.Load()
	if err != nil {
		return monitorResources{}, fmt.Errorf("load config: %w", err)
	}

	postgresPool, err := database.NewPostgres(ctx, config.Database)
	if err != nil {
		return monitorResources{}, fmt.Errorf("connect postgres: %w", err)
	}

	redisClient, err := redisclient.New(ctx, config.Redis)
	if err != nil {
		postgresPool.Close()
		return monitorResources{}, fmt.Errorf("connect redis: %w", err)
	}

	adminRepository := repository.NewPostgresAdminRepository(postgresPool)
	passwordHasher := security.NewBCryptPasswordHasher(config.Security.BCryptCost)
	service := adminapp.NewService(adminRepository, passwordHasher)

	closeResources := func() {
		_ = redisClient.Close()
		postgresPool.Close()
	}

	return monitorResources{
		config:      config,
		service:     service,
		redisClient: redisClient,
		close:       closeResources,
	}, nil
}

func collectMonitorSnapshot(ctx context.Context, resources monitorResources, metricsURL string, previousRequests *metricSample) monitorSnapshot {
	collectedAt := time.Now()
	snapshot := monitorSnapshot{collectedAt: collectedAt}

	databaseContext, cancelDatabase := context.WithTimeout(ctx, 3*time.Second)
	databaseSnapshot, err := resources.service.GetMonitorDatabaseSnapshot(databaseContext)
	cancelDatabase()
	if err != nil {
		snapshot.databaseError = err
	} else {
		snapshot.database = databaseSnapshot
	}

	redisContext, cancelRedis := context.WithTimeout(ctx, 3*time.Second)
	redisOnlineDrivers, err := countRedisKeys(redisContext, resources.redisClient, "driver:online:*")
	cancelRedis()
	if err != nil {
		snapshot.redisError = err
	} else {
		snapshot.redisConnected = true
		snapshot.redisOnlineDrivers = redisOnlineDrivers
	}

	metricsContext, cancelMetrics := context.WithTimeout(ctx, 3*time.Second)
	httpRequestsTotal, err := fetchHTTPRequestsTotal(metricsContext, metricsURL)
	cancelMetrics()
	if err != nil {
		snapshot.metricsError = err
	} else {
		snapshot.metricsConnected = true
		snapshot.httpRequestsTotal = httpRequestsTotal
		if previousRequests != nil && collectedAt.After(previousRequests.collectedAt) && httpRequestsTotal >= previousRequests.value {
			minutes := collectedAt.Sub(previousRequests.collectedAt).Minutes()
			if minutes > 0 {
				rate := (httpRequestsTotal - previousRequests.value) / minutes
				snapshot.httpRequestsPerMinute = &rate
			}
		}
	}

	return snapshot
}

type metricSample struct {
	value       float64
	collectedAt time.Time
}

func countRedisKeys(ctx context.Context, client *goredis.Client, pattern string) (int64, error) {
	var cursor uint64
	var count int64
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return 0, fmt.Errorf("scan redis keys %s: %w", pattern, err)
		}
		count += int64(len(keys))
		cursor = nextCursor
		if cursor == 0 {
			return count, nil
		}
	}
}

func fetchHTTPRequestsTotal(ctx context.Context, metricsURL string) (float64, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, metricsURL, nil)
	if err != nil {
		return 0, fmt.Errorf("create metrics request: %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("fetch metrics: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, response.Body)
		return 0, fmt.Errorf("fetch metrics: unexpected status %d", response.StatusCode)
	}

	return parseHTTPRequestsTotal(response.Body)
}

func parseHTTPRequestsTotal(reader io.Reader) (float64, error) {
	scanner := bufio.NewScanner(reader)
	total := 0.0
	found := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		matches := requestMetricPattern.FindStringSubmatch(line)
		if len(matches) != 2 {
			continue
		}
		value, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0, fmt.Errorf("parse request metric %q: %w", matches[1], err)
		}
		total += value
		found = true
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("read metrics: %w", err)
	}
	if !found {
		return 0, errors.New("taxi_http_requests_total metric not found")
	}

	return total, nil
}

func renderMonitor(writer io.Writer, snapshot monitorSnapshot, options monitorOptions) {
	if !options.once {
		fmt.Fprint(writer, "\x1b[H")
	}

	lines := monitorLines(snapshot, options)
	if len(lines) > options.height {
		lines = lines[:options.height]
	}

	for _, line := range lines {
		fmt.Fprintln(writer, fitLine(line, options.width))
	}
}

func monitorLines(snapshot monitorSnapshot, options monitorOptions) []string {
	requestsPerMinute := "warming up"
	if snapshot.httpRequestsPerMinute != nil {
		requestsPerMinute = fmt.Sprintf("%.1f/min", *snapshot.httpRequestsPerMinute)
	}

	databaseStatus := "ok"
	if snapshot.databaseError != nil {
		databaseStatus = snapshot.databaseError.Error()
	}
	redisStatus := "ok"
	if !snapshot.redisConnected {
		redisStatus = errorText(snapshot.redisError)
	}
	metricsStatus := "ok"
	if !snapshot.metricsConnected {
		metricsStatus = errorText(snapshot.metricsError)
	}

	width := options.width
	separator := strings.Repeat("-", width)
	lines := []string{
		separator,
		centerText("Taxi Platform Console Monitor", width),
		separator,
		fmt.Sprintf("Collected: %s | Refresh: %s | Frame: %dx%d | Ctrl+C to exit", snapshot.collectedAt.Format(time.RFC3339), options.interval, options.width, options.height),
	}
	lines = append(lines, renderTable("SYSTEM", []string{"Component", "Status"}, [][]string{
		{"Database", databaseStatus},
		{"Redis", redisStatus},
		{"Metrics", metricsStatus},
	}, width)...)
	lines = append(lines, renderTable("SUMMARY", []string{"Area", "Metric", "Value"}, [][]string{
		{"Users", "total", formatInt(snapshot.database.TotalUsers)},
		{"Users", "active", formatInt(snapshot.database.ActiveUsers)},
		{"Users", "active 15m", formatInt(snapshot.database.RecentlyActiveUsers)},
		{"Taxi parks", "total", formatInt(snapshot.database.TotalTaxiParks)},
		{"Taxi parks", "active verified", formatInt(snapshot.database.ActiveTaxiParks)},
		{"Drivers", "total", formatInt(snapshot.database.TotalDrivers)},
		{"Drivers", "online DB", formatInt(snapshot.database.OnlineDrivers)},
		{"Drivers", "online Redis", formatInt(snapshot.redisOnlineDrivers)},
		{"Drivers", "busy", formatInt(snapshot.database.BusyDrivers)},
		{"Drivers", "blocked", formatInt(snapshot.database.BlockedDrivers)},
	}, width)...)
	lines = append(lines, renderTable("ORDERS", []string{"Metric", "Value"}, [][]string{
		{"total", formatInt(snapshot.database.TotalOrders)},
		{"active", formatInt(snapshot.database.ActiveOrders)},
		{"searching", formatInt(snapshot.database.SearchingOrders)},
		{"assigned/arriving/waiting", formatInt(snapshot.database.AssignedOrders)},
		{"in progress", formatInt(snapshot.database.InProgressOrders)},
		{"completed today", formatInt(snapshot.database.CompletedOrdersToday)},
		{"cancelled today", formatInt(snapshot.database.CancelledOrdersToday)},
		{"failed today", formatInt(snapshot.database.FailedOrdersToday)},
	}, width)...)
	lines = append(lines, renderTable("HTTP", []string{"Metric", "Value"}, [][]string{
		{"requests total", formatFloat(snapshot.httpRequestsTotal)},
		{"requests per minute", requestsPerMinute},
		{"metrics URL", options.metricsURL},
	}, width)...)
	lines = append(lines,
		separator,
		"Notes: Redis online = driver:online:* keys. Active users 15m = active refresh tokens approximation.",
		separator,
	)

	return lines
}

func renderTable(title string, headers []string, rows [][]string, maxWidth int) []string {
	if maxWidth < 20 {
		maxWidth = 20
	}
	columnWidths := make([]int, len(headers))
	for index, header := range headers {
		columnWidths[index] = len([]rune(header))
	}
	for _, row := range rows {
		for index := range headers {
			value := ""
			if index < len(row) {
				value = row[index]
			}
			if length := len([]rune(value)); length > columnWidths[index] {
				columnWidths[index] = length
			}
		}
	}

	const maxColumnWidth = 42
	for index := range columnWidths {
		if columnWidths[index] > maxColumnWidth {
			columnWidths[index] = maxColumnWidth
		}
	}

	border := tableBorder(columnWidths)
	lines := []string{"", title, border, tableRow(headers, columnWidths), border}
	for _, row := range rows {
		cells := make([]string, len(headers))
		for index := range headers {
			if index < len(row) {
				cells[index] = row[index]
			}
		}
		lines = append(lines, tableRow(cells, columnWidths))
	}
	lines = append(lines, border)
	for index, line := range lines {
		lines[index] = fitLine(line, maxWidth)
	}
	return lines
}

func tableBorder(widths []int) string {
	var builder strings.Builder
	builder.WriteString("+")
	for _, width := range widths {
		builder.WriteString(strings.Repeat("-", width+2))
		builder.WriteString("+")
	}
	return builder.String()
}

func tableRow(values []string, widths []int) string {
	var builder strings.Builder
	builder.WriteString("|")
	for index, width := range widths {
		value := ""
		if index < len(values) {
			value = values[index]
		}
		value = fitCell(value, width)
		builder.WriteString(" ")
		builder.WriteString(value)
		builder.WriteString(" |")
	}
	return builder.String()
}

func fitCell(value string, width int) string {
	runes := []rune(value)
	if len(runes) > width {
		if width <= 1 {
			return string(runes[:width])
		}
		return string(runes[:width-1]) + ">"
	}
	return string(runes) + strings.Repeat(" ", width-len(runes))
}

func enterMonitorScreen(writer io.Writer) {
	fmt.Fprint(writer, "\x1b[?1049h\x1b[?25l\x1b[2J\x1b[H")
}

func leaveMonitorScreen(writer io.Writer) {
	fmt.Fprint(writer, "\x1b[?25h\x1b[?1049l")
}

func fitLine(line string, width int) string {
	if width <= 0 {
		return line
	}
	runes := []rune(line)
	if len(runes) > width {
		if width <= 1 {
			return string(runes[:width])
		}
		return string(runes[:width-1]) + ">"
	}
	return string(runes) + strings.Repeat(" ", width-len(runes))
}

func centerText(text string, width int) string {
	textLength := len([]rune(text))
	if textLength >= width {
		return text
	}
	leftPadding := int(math.Floor(float64(width-textLength) / 2))
	return strings.Repeat(" ", leftPadding) + text
}

func errorText(err error) string {
	if err == nil {
		return "unknown error"
	}
	return err.Error()
}

func formatFloat(value float64) string {
	if value == 0 {
		return "0"
	}
	if value == math.Trunc(value) {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.2f", value)
}

func formatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}

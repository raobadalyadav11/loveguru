package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"loveguru/internal/logger"
)

// DatabaseMetrics tracks database performance and health metrics
type DatabaseMetrics struct {
	mu sync.RWMutex

	// Connection metrics
	ActiveConnections int64
	IdleConnections   int64

	// Query metrics
	TotalQueries     int64
	FailedQueries    int64
	AverageQueryTime float64
	MaxQueryTime     float64
	MinQueryTime     float64

	// Error metrics
	ConstraintViolations int64
	ConnectionErrors     int64
	TimeoutErrors        int64

	// Transaction metrics
	TotalTransactions      int64
	FailedTransactions     int64
	AverageTransactionTime float64

	// Last health check
	LastHealthCheck time.Time
	IsHealthy       bool
	HealthMessage   string
}

// DatabaseMonitor provides comprehensive database monitoring
type DatabaseMonitor struct {
	db      *sql.DB
	metrics *DatabaseMetrics
	logger  logger.Logger
	ticker  *time.Ticker
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewDatabaseMonitor creates a new database monitor
func NewDatabaseMonitor(db *sql.DB, logger logger.Logger) *DatabaseMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	monitor := &DatabaseMonitor{
		db:      db,
		metrics: &DatabaseMetrics{},
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
		ticker:  time.NewTicker(30 * time.Second), // Check every 30 seconds
	}

	go monitor.startMonitoring()
	return monitor
}

// StartMonitoring begins the monitoring process
func (dm *DatabaseMonitor) startMonitoring() {
	// Initial health check
	dm.performHealthCheck()

	for {
		select {
		case <-dm.ticker.C:
			dm.performHealthCheck()
		case <-dm.ctx.Done():
			dm.ticker.Stop()
			return
		}
	}
}

// StopMonitoring stops the monitoring process
func (dm *DatabaseMonitor) StopMonitoring() {
	dm.cancel()
	dm.ticker.Stop()
}

// performHealthCheck performs a comprehensive health check
func (dm *DatabaseMonitor) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dm.metrics.mu.Lock()
	defer dm.metrics.mu.Unlock()

	dm.metrics.LastHealthCheck = time.Now()

	// Test database connection
	if err := dm.db.PingContext(ctx); err != nil {
		dm.metrics.IsHealthy = false
		dm.metrics.HealthMessage = fmt.Sprintf("Database connection failed: %v", err)
		dm.metrics.ConnectionErrors++
		dm.logger.Error(ctx, "Database health check failed", err)
		return
	}

	// Get connection statistics - using correct field names
	stats := dm.db.Stats()
	dm.metrics.ActiveConnections = int64(stats.InUse)
	dm.metrics.IdleConnections = int64(stats.Idle)

	// Check connection pool health (simplified)
	if stats.InUse > 20 { // Arbitrary threshold
		dm.metrics.HealthMessage = fmt.Sprintf("High connection usage: %d active", stats.InUse)
		dm.logger.Warn(ctx, "Database connection pool high usage",
			"active", stats.InUse)
	}

	// Mark as healthy if connection is working
	dm.metrics.IsHealthy = true
	dm.metrics.HealthMessage = "Database is healthy"

	dm.logger.Info(ctx, "Database health check passed",
		"active_connections", stats.InUse,
		"idle_connections", stats.Idle)
}

// RecordQuery records a database query execution
func (dm *DatabaseMonitor) RecordQuery(success bool, duration time.Duration) {
	dm.metrics.mu.Lock()
	defer dm.metrics.mu.Unlock()

	dm.metrics.TotalQueries++
	if !success {
		dm.metrics.FailedQueries++
	}

	// Update timing statistics
	if dm.metrics.TotalQueries == 1 {
		dm.metrics.MinQueryTime = duration.Seconds()
		dm.metrics.MaxQueryTime = duration.Seconds()
		dm.metrics.AverageQueryTime = duration.Seconds()
	} else {
		// Update min/max
		if duration.Seconds() < dm.metrics.MinQueryTime {
			dm.metrics.MinQueryTime = duration.Seconds()
		}
		if duration.Seconds() > dm.metrics.MaxQueryTime {
			dm.metrics.MaxQueryTime = duration.Seconds()
		}

		// Update average (simple moving average)
		dm.metrics.AverageQueryTime = (dm.metrics.AverageQueryTime + duration.Seconds()) / 2
	}

	// Log slow queries
	if duration > 1*time.Second {
		dm.logger.Warn(dm.ctx, "Slow database query detected",
			"duration_ms", duration.Milliseconds(),
			"success", success)
	}
}

// RecordTransaction records a transaction execution
func (dm *DatabaseMonitor) RecordTransaction(success bool, duration time.Duration) {
	dm.metrics.mu.Lock()
	defer dm.metrics.mu.Unlock()

	dm.metrics.TotalTransactions++
	if !success {
		dm.metrics.FailedTransactions++
	}

	// Update average transaction time
	if dm.metrics.TotalTransactions == 1 {
		dm.metrics.AverageTransactionTime = duration.Seconds()
	} else {
		dm.metrics.AverageTransactionTime = (dm.metrics.AverageTransactionTime + duration.Seconds()) / 2
	}

	// Log failed transactions
	if !success {
		dm.logger.Error(dm.ctx, "Database transaction failed", fmt.Errorf("transaction failed"))
	}
}

// RecordConstraintViolation records a constraint violation
func (dm *DatabaseMonitor) RecordConstraintViolation() {
	dm.metrics.mu.Lock()
	defer dm.metrics.mu.Unlock()

	dm.metrics.ConstraintViolations++
	dm.logger.Warn(dm.ctx, "Database constraint violation detected")
}

// GetMetrics returns the current database metrics
func (dm *DatabaseMonitor) GetMetrics() *DatabaseMetrics {
	dm.metrics.mu.RLock()
	defer dm.metrics.mu.RUnlock()

	// Return a copy to avoid race conditions
	metrics := *dm.metrics
	return &metrics
}

// GetHealthStatus returns the current health status
func (dm *DatabaseMonitor) GetHealthStatus() (bool, string, time.Time) {
	dm.metrics.mu.RLock()
	defer dm.metrics.mu.RUnlock()

	return dm.metrics.IsHealthy, dm.metrics.HealthMessage, dm.metrics.LastHealthCheck
}

// CheckConnectionHealth performs a one-time connection health check
func CheckConnectionHealth(ctx context.Context, db *sql.DB, logger logger.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Error(ctx, "Database connection health check failed", err)
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Check if we can perform a simple query
	var result int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
		logger.Error(ctx, "Database query health check failed", err)
		return fmt.Errorf("database query failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected query result: %d", result)
	}

	logger.Info(ctx, "Database health check passed")
	return nil
}

// MonitoredQuery wraps a database query with monitoring
func MonitoredQuery(ctx context.Context, monitor *DatabaseMonitor, operation string, queryFunc func() error) error {
	start := time.Now()

	err := queryFunc()
	duration := time.Since(start)

	success := err == nil
	monitor.RecordQuery(success, duration)

	if !success {
		monitor.logger.Error(ctx, fmt.Sprintf("Database operation failed: %s", operation), err,
			"duration_ms", duration.Milliseconds())
	}

	return err
}

// MonitoredTransaction wraps a database transaction with monitoring
func MonitoredTransaction(ctx context.Context, monitor *DatabaseMonitor, db *sql.DB, txFunc func(*Queries) error) error {
	start := time.Now()

	err := Transaction(ctx, db, txFunc)
	duration := time.Since(start)

	success := err == nil
	monitor.RecordTransaction(success, duration)

	if !success {
		monitor.logger.Error(ctx, "Database transaction failed", err,
			"duration_ms", duration.Milliseconds())
	}

	return err
}

// GetDatabaseStatus returns comprehensive database status information
func GetDatabaseStatus(ctx context.Context, db *sql.DB, monitor *DatabaseMonitor) map[string]interface{} {
	status := make(map[string]interface{})

	// Connection health
	if err := CheckConnectionHealth(ctx, db, monitor.logger); err != nil {
		status["healthy"] = false
		status["error"] = err.Error()
	} else {
		status["healthy"] = true
	}

	// Connection statistics - using correct field names
	stats := db.Stats()
	status["connections"] = map[string]interface{}{
		"active":        stats.InUse,
		"idle":          stats.Idle,
		"wait_count":    stats.WaitCount,
		"wait_duration": stats.WaitDuration.String(),
	}

	// Monitor metrics
	if monitor != nil {
		metrics := monitor.GetMetrics()
		status["metrics"] = map[string]interface{}{
			"total_queries":         metrics.TotalQueries,
			"failed_queries":        metrics.FailedQueries,
			"success_rate":          calculateSuccessRate(metrics.TotalQueries, metrics.FailedQueries),
			"average_query_time":    fmt.Sprintf("%.2fms", metrics.AverageQueryTime*1000),
			"max_query_time":        fmt.Sprintf("%.2fms", metrics.MaxQueryTime*1000),
			"total_transactions":    metrics.TotalTransactions,
			"failed_transactions":   metrics.FailedTransactions,
			"constraint_violations": metrics.ConstraintViolations,
			"last_health_check":     metrics.LastHealthCheck.Format(time.RFC3339),
		}
	}

	return status
}

// calculateSuccessRate calculates the success rate percentage
func calculateSuccessRate(total, failed int64) float64 {
	if total == 0 {
		return 100.0
	}
	return float64(total-failed) / float64(total) * 100
}

// Example usage
func ExampleMonitoring() {
	// This would typically be called from main.go
	log.Println("Database monitoring system is ready")
}

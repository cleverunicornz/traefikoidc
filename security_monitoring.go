package traefikoidc

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SecurityEvent represents a security-related event that should be logged and monitored
type SecurityEvent struct {
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Timestamp   time.Time              `json:"timestamp"`
	ClientIP    string                 `json:"client_ip"`
	UserAgent   string                 `json:"user_agent"`
	RequestPath string                 `json:"request_path"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// SecurityMonitor tracks security events and suspicious activity patterns
type SecurityMonitor struct {
	// Event counters
	authFailures         int64
	tokenValidationFails int64
	rateLimitHits        int64
	suspiciousRequests   int64

	// IP-based tracking
	ipFailures map[string]*IPFailureTracker
	ipMutex    sync.RWMutex

	// Pattern detection
	patternDetector *SuspiciousPatternDetector

	// Event handlers
	eventHandlers []SecurityEventHandler

	// Configuration
	config SecurityMonitorConfig

	// Logger
	logger *Logger
}

// IPFailureTracker tracks failures for a specific IP address
type IPFailureTracker struct {
	FailureCount int64
	LastFailure  time.Time
	FirstFailure time.Time
	FailureTypes map[string]int64
	IsBlocked    bool
	BlockedUntil time.Time
	mutex        sync.RWMutex
}

// SuspiciousPatternDetector identifies patterns that may indicate attacks
type SuspiciousPatternDetector struct {
	// Time-based windows for pattern detection
	shortWindow  time.Duration // 1 minute
	mediumWindow time.Duration // 5 minutes
	longWindow   time.Duration // 15 minutes

	// Pattern thresholds
	rapidFailureThreshold      int // failures in short window
	distributedAttackThreshold int // failures across IPs in medium window
	persistentAttackThreshold  int // failures in long window

	// Pattern tracking
	recentEvents []SecurityEvent
	eventsMutex  sync.RWMutex
}

// SecurityEventHandler defines the interface for handling security events
type SecurityEventHandler interface {
	HandleSecurityEvent(event SecurityEvent)
}

// SecurityMonitorConfig contains configuration for the security monitor
type SecurityMonitorConfig struct {
	// Failure thresholds
	MaxFailuresPerIP     int `json:"max_failures_per_ip"`
	FailureWindowMinutes int `json:"failure_window_minutes"`
	BlockDurationMinutes int `json:"block_duration_minutes"`

	// Pattern detection settings
	EnablePatternDetection bool `json:"enable_pattern_detection"`
	RapidFailureThreshold  int  `json:"rapid_failure_threshold"`

	// Monitoring settings
	EnableDetailedLogging bool `json:"enable_detailed_logging"`
	LogSuspiciousOnly     bool `json:"log_suspicious_only"`

	// Cleanup settings
	CleanupIntervalMinutes int `json:"cleanup_interval_minutes"`
	RetentionHours         int `json:"retention_hours"`
}

// DefaultSecurityMonitorConfig returns a default configuration
func DefaultSecurityMonitorConfig() SecurityMonitorConfig {
	return SecurityMonitorConfig{
		MaxFailuresPerIP:       10,
		FailureWindowMinutes:   15,
		BlockDurationMinutes:   60,
		EnablePatternDetection: true,
		RapidFailureThreshold:  5,
		EnableDetailedLogging:  true,
		LogSuspiciousOnly:      false,
		CleanupIntervalMinutes: 30,
		RetentionHours:         24,
	}
}

// NewSecurityMonitor creates a new security monitor instance
func NewSecurityMonitor(config SecurityMonitorConfig, logger *Logger) *SecurityMonitor {
	sm := &SecurityMonitor{
		ipFailures:      make(map[string]*IPFailureTracker),
		eventHandlers:   make([]SecurityEventHandler, 0),
		config:          config,
		logger:          logger,
		patternDetector: NewSuspiciousPatternDetector(),
	}

	// Start cleanup routine
	go sm.startCleanupRoutine()

	return sm
}

// NewSuspiciousPatternDetector creates a new pattern detector
func NewSuspiciousPatternDetector() *SuspiciousPatternDetector {
	return &SuspiciousPatternDetector{
		shortWindow:                1 * time.Minute,
		mediumWindow:               5 * time.Minute,
		longWindow:                 15 * time.Minute,
		rapidFailureThreshold:      5,
		distributedAttackThreshold: 20,
		persistentAttackThreshold:  50,
		recentEvents:               make([]SecurityEvent, 0),
	}
}

// RecordAuthenticationFailure records an authentication failure event
func (sm *SecurityMonitor) RecordAuthenticationFailure(clientIP, userAgent, requestPath, reason string, details map[string]interface{}) {
	atomic.AddInt64(&sm.authFailures, 1)

	event := SecurityEvent{
		Type:        "authentication_failure",
		Severity:    "medium",
		Timestamp:   time.Now(),
		ClientIP:    clientIP,
		UserAgent:   userAgent,
		RequestPath: requestPath,
		Message:     fmt.Sprintf("Authentication failed: %s", reason),
		Details:     details,
	}

	sm.recordIPFailure(clientIP, "auth_failure")
	sm.processSecurityEvent(event)
}

// RecordTokenValidationFailure records a token validation failure
func (sm *SecurityMonitor) RecordTokenValidationFailure(clientIP, userAgent, requestPath, reason string, tokenPrefix string) {
	atomic.AddInt64(&sm.tokenValidationFails, 1)

	details := map[string]interface{}{
		"reason": reason,
	}
	if tokenPrefix != "" {
		details["token_prefix"] = tokenPrefix
	}

	event := SecurityEvent{
		Type:        "token_validation_failure",
		Severity:    "medium",
		Timestamp:   time.Now(),
		ClientIP:    clientIP,
		UserAgent:   userAgent,
		RequestPath: requestPath,
		Message:     fmt.Sprintf("Token validation failed: %s", reason),
		Details:     details,
	}

	sm.recordIPFailure(clientIP, "token_failure")
	sm.processSecurityEvent(event)
}

// RecordRateLimitHit records when rate limiting is triggered
func (sm *SecurityMonitor) RecordRateLimitHit(clientIP, userAgent, requestPath string) {
	atomic.AddInt64(&sm.rateLimitHits, 1)

	event := SecurityEvent{
		Type:        "rate_limit_hit",
		Severity:    "low",
		Timestamp:   time.Now(),
		ClientIP:    clientIP,
		UserAgent:   userAgent,
		RequestPath: requestPath,
		Message:     "Rate limit exceeded",
		Details: map[string]interface{}{
			"limit_type": "token_verification",
		},
	}

	sm.recordIPFailure(clientIP, "rate_limit")
	sm.processSecurityEvent(event)
}

// RecordSuspiciousActivity records suspicious activity that doesn't fit other categories
func (sm *SecurityMonitor) RecordSuspiciousActivity(clientIP, userAgent, requestPath, activityType, description string, details map[string]interface{}) {
	atomic.AddInt64(&sm.suspiciousRequests, 1)

	event := SecurityEvent{
		Type:        "suspicious_activity",
		Severity:    "high",
		Timestamp:   time.Now(),
		ClientIP:    clientIP,
		UserAgent:   userAgent,
		RequestPath: requestPath,
		Message:     fmt.Sprintf("Suspicious activity detected: %s - %s", activityType, description),
		Details:     details,
	}

	sm.recordIPFailure(clientIP, "suspicious")
	sm.processSecurityEvent(event)
}

// recordIPFailure tracks failures for a specific IP address
func (sm *SecurityMonitor) recordIPFailure(clientIP, failureType string) {
	sm.ipMutex.Lock()
	defer sm.ipMutex.Unlock()

	tracker, exists := sm.ipFailures[clientIP]
	if !exists {
		tracker = &IPFailureTracker{
			FailureTypes: make(map[string]int64),
			FirstFailure: time.Now(),
		}
		sm.ipFailures[clientIP] = tracker
	}

	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	tracker.FailureCount++
	tracker.LastFailure = time.Now()
	tracker.FailureTypes[failureType]++

	// Check if IP should be blocked
	windowStart := time.Now().Add(-time.Duration(sm.config.FailureWindowMinutes) * time.Minute)
	if tracker.FirstFailure.After(windowStart) && tracker.FailureCount >= int64(sm.config.MaxFailuresPerIP) {
		if !tracker.IsBlocked {
			tracker.IsBlocked = true
			tracker.BlockedUntil = time.Now().Add(time.Duration(sm.config.BlockDurationMinutes) * time.Minute)

			sm.logger.Errorf("IP %s blocked due to %d failures (types: %v)", clientIP, tracker.FailureCount, tracker.FailureTypes)

			// Record blocking event
			blockEvent := SecurityEvent{
				Type:      "ip_blocked",
				Severity:  "high",
				Timestamp: time.Now(),
				ClientIP:  clientIP,
				Message:   fmt.Sprintf("IP blocked due to %d failures in %d minutes", tracker.FailureCount, sm.config.FailureWindowMinutes),
				Details: map[string]interface{}{
					"failure_count": tracker.FailureCount,
					"failure_types": tracker.FailureTypes,
					"blocked_until": tracker.BlockedUntil,
				},
			}
			sm.processSecurityEvent(blockEvent)
		}
	}
}

// IsIPBlocked checks if an IP address is currently blocked
func (sm *SecurityMonitor) IsIPBlocked(clientIP string) bool {
	sm.ipMutex.RLock()
	defer sm.ipMutex.RUnlock()

	tracker, exists := sm.ipFailures[clientIP]
	if !exists {
		return false
	}

	tracker.mutex.RLock()
	defer tracker.mutex.RUnlock()

	if tracker.IsBlocked && time.Now().Before(tracker.BlockedUntil) {
		return true
	}

	// Unblock if time has passed
	if tracker.IsBlocked && time.Now().After(tracker.BlockedUntil) {
		tracker.IsBlocked = false
		sm.logger.Infof("IP %s automatically unblocked", clientIP)
	}

	return false
}

// processSecurityEvent processes a security event through all handlers and pattern detection
func (sm *SecurityMonitor) processSecurityEvent(event SecurityEvent) {
	// Add to pattern detector
	if sm.config.EnablePatternDetection {
		sm.patternDetector.AddEvent(event)

		// Check for suspicious patterns
		if patterns := sm.patternDetector.DetectSuspiciousPatterns(); len(patterns) > 0 {
			for _, pattern := range patterns {
				sm.logger.Errorf("Suspicious pattern detected: %s", pattern)

				patternEvent := SecurityEvent{
					Type:      "suspicious_pattern",
					Severity:  "high",
					Timestamp: time.Now(),
					Message:   fmt.Sprintf("Suspicious pattern detected: %s", pattern),
					Details: map[string]interface{}{
						"pattern_type":  pattern,
						"trigger_event": event,
					},
				}
				sm.handleSecurityEvent(patternEvent)
			}
		}
	}

	sm.handleSecurityEvent(event)
}

// handleSecurityEvent sends the event to all registered handlers
func (sm *SecurityMonitor) handleSecurityEvent(event SecurityEvent) {
	// Log the event
	if sm.config.EnableDetailedLogging && (!sm.config.LogSuspiciousOnly || event.Severity == "high") {
		sm.logger.Infof("Security Event [%s/%s]: %s (IP: %s, Path: %s)",
			event.Type, event.Severity, event.Message, event.ClientIP, event.RequestPath)
	}

	// Send to all handlers
	for _, handler := range sm.eventHandlers {
		go handler.HandleSecurityEvent(event)
	}
}

// AddEventHandler adds a security event handler
func (sm *SecurityMonitor) AddEventHandler(handler SecurityEventHandler) {
	sm.eventHandlers = append(sm.eventHandlers, handler)
}

// GetSecurityMetrics returns current security metrics
func (sm *SecurityMonitor) GetSecurityMetrics() map[string]interface{} {
	sm.ipMutex.RLock()
	defer sm.ipMutex.RUnlock()

	blockedIPs := 0
	totalTrackedIPs := len(sm.ipFailures)

	for _, tracker := range sm.ipFailures {
		tracker.mutex.RLock()
		if tracker.IsBlocked && time.Now().Before(tracker.BlockedUntil) {
			blockedIPs++
		}
		tracker.mutex.RUnlock()
	}

	return map[string]interface{}{
		"auth_failures":          atomic.LoadInt64(&sm.authFailures),
		"token_validation_fails": atomic.LoadInt64(&sm.tokenValidationFails),
		"rate_limit_hits":        atomic.LoadInt64(&sm.rateLimitHits),
		"suspicious_requests":    atomic.LoadInt64(&sm.suspiciousRequests),
		"blocked_ips":            blockedIPs,
		"tracked_ips":            totalTrackedIPs,
		"uptime_hours":           time.Since(time.Now().Add(-24 * time.Hour)).Hours(), // Placeholder
	}
}

// AddEvent adds an event to the pattern detector
func (spd *SuspiciousPatternDetector) AddEvent(event SecurityEvent) {
	spd.eventsMutex.Lock()
	defer spd.eventsMutex.Unlock()

	spd.recentEvents = append(spd.recentEvents, event)

	// Clean old events
	cutoff := time.Now().Add(-spd.longWindow)
	var filteredEvents []SecurityEvent
	for _, e := range spd.recentEvents {
		if e.Timestamp.After(cutoff) {
			filteredEvents = append(filteredEvents, e)
		}
	}
	spd.recentEvents = filteredEvents
}

// DetectSuspiciousPatterns analyzes recent events for suspicious patterns
func (spd *SuspiciousPatternDetector) DetectSuspiciousPatterns() []string {
	spd.eventsMutex.RLock()
	defer spd.eventsMutex.RUnlock()

	var patterns []string
	now := time.Now()

	// Check for rapid failures from single IP
	ipCounts := make(map[string]int)
	shortWindowStart := now.Add(-spd.shortWindow)

	for _, event := range spd.recentEvents {
		if event.Timestamp.After(shortWindowStart) &&
			(event.Type == "authentication_failure" || event.Type == "token_validation_failure") {
			ipCounts[event.ClientIP]++
		}
	}

	for ip, count := range ipCounts {
		if count >= spd.rapidFailureThreshold {
			patterns = append(patterns, fmt.Sprintf("rapid_failures_from_ip_%s", ip))
		}
	}

	// Check for distributed attack (many IPs failing)
	mediumWindowStart := now.Add(-spd.mediumWindow)
	uniqueFailingIPs := make(map[string]bool)

	for _, event := range spd.recentEvents {
		if event.Timestamp.After(mediumWindowStart) &&
			(event.Type == "authentication_failure" || event.Type == "token_validation_failure") {
			uniqueFailingIPs[event.ClientIP] = true
		}
	}

	if len(uniqueFailingIPs) >= spd.distributedAttackThreshold {
		patterns = append(patterns, "distributed_attack_pattern")
	}

	// Check for persistent attack
	longWindowStart := now.Add(-spd.longWindow)
	persistentFailures := 0

	for _, event := range spd.recentEvents {
		if event.Timestamp.After(longWindowStart) &&
			(event.Type == "authentication_failure" || event.Type == "token_validation_failure") {
			persistentFailures++
		}
	}

	if persistentFailures >= spd.persistentAttackThreshold {
		patterns = append(patterns, "persistent_attack_pattern")
	}

	return patterns
}

// startCleanupRoutine starts the background cleanup routine
func (sm *SecurityMonitor) startCleanupRoutine() {
	ticker := time.NewTicker(time.Duration(sm.config.CleanupIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.cleanup()
	}
}

// cleanup removes old tracking data
func (sm *SecurityMonitor) cleanup() {
	sm.ipMutex.Lock()
	defer sm.ipMutex.Unlock()

	cutoff := time.Now().Add(-time.Duration(sm.config.RetentionHours) * time.Hour)

	for ip, tracker := range sm.ipFailures {
		tracker.mutex.RLock()
		shouldRemove := tracker.LastFailure.Before(cutoff) && !tracker.IsBlocked
		tracker.mutex.RUnlock()

		if shouldRemove {
			delete(sm.ipFailures, ip)
		}
	}

	sm.logger.Debugf("Security monitor cleanup completed, tracking %d IPs", len(sm.ipFailures))
}

// ExtractClientIP extracts the client IP from the request, considering proxy headers
func ExtractClientIP(r *http.Request) string {
	// Check X-Real-IP header first (highest priority)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// Check X-Forwarded-For header second
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// LoggingSecurityEventHandler logs security events to the standard logger
type LoggingSecurityEventHandler struct {
	logger *Logger
}

// NewLoggingSecurityEventHandler creates a new logging event handler
func NewLoggingSecurityEventHandler(logger *Logger) *LoggingSecurityEventHandler {
	return &LoggingSecurityEventHandler{logger: logger}
}

// HandleSecurityEvent implements SecurityEventHandler
func (h *LoggingSecurityEventHandler) HandleSecurityEvent(event SecurityEvent) {
	switch event.Severity {
	case "high":
		h.logger.Errorf("SECURITY [%s]: %s (IP: %s)", event.Type, event.Message, event.ClientIP)
	case "medium":
		h.logger.Errorf("SECURITY [%s]: %s (IP: %s)", event.Type, event.Message, event.ClientIP)
	case "low":
		h.logger.Infof("SECURITY [%s]: %s (IP: %s)", event.Type, event.Message, event.ClientIP)
	default:
		h.logger.Debugf("SECURITY [%s]: %s (IP: %s)", event.Type, event.Message, event.ClientIP)
	}
}

// MetricsSecurityEventHandler tracks security metrics
type MetricsSecurityEventHandler struct {
	eventCounts map[string]int64
	mutex       sync.RWMutex
}

// NewMetricsSecurityEventHandler creates a new metrics event handler
func NewMetricsSecurityEventHandler() *MetricsSecurityEventHandler {
	return &MetricsSecurityEventHandler{
		eventCounts: make(map[string]int64),
	}
}

// HandleSecurityEvent implements SecurityEventHandler
func (h *MetricsSecurityEventHandler) HandleSecurityEvent(event SecurityEvent) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.eventCounts[event.Type]++
	h.eventCounts[fmt.Sprintf("%s_%s", event.Type, event.Severity)]++
}

// GetMetrics returns the current metrics
func (h *MetricsSecurityEventHandler) GetMetrics() map[string]int64 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	metrics := make(map[string]int64)
	for k, v := range h.eventCounts {
		metrics[k] = v
	}
	return metrics
}

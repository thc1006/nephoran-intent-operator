package providers

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// LDAPConnectionPool manages a pool of LDAP connections for improved performance
type LDAPConnectionPool struct {
	config      *LDAPConfig
	logger      *slog.Logger
	connections chan *ldap.Conn
	mu          sync.RWMutex
	closed      bool
	
	// Connection management
	activeConnections int
	maxConnections    int
	minConnections    int
	
	// Health monitoring
	healthChecker   *LDAPHealthChecker
	lastHealthCheck time.Time
	
	// Statistics
	stats *LDAPPoolStats
}

// LDAPPoolStats tracks connection pool statistics
type LDAPPoolStats struct {
	mu                 sync.RWMutex
	TotalConnections   int64     `json:"total_connections"`
	ActiveConnections  int       `json:"active_connections"`
	IdleConnections    int       `json:"idle_connections"`
	ConnectionsCreated int64     `json:"connections_created"`
	ConnectionsReused  int64     `json:"connections_reused"`
	ConnectionsFailed  int64     `json:"connections_failed"`
	HealthChecksPassed int64     `json:"health_checks_passed"`
	HealthChecksFailed int64     `json:"health_checks_failed"`
	LastHealthCheck    time.Time `json:"last_health_check"`
	AverageCheckTime   time.Duration `json:"average_check_time"`
}

// LDAPHealthChecker monitors LDAP server health
type LDAPHealthChecker struct {
	config    *LDAPConfig
	logger    *slog.Logger
	interval  time.Duration
	timeout   time.Duration
	lastCheck time.Time
	healthy   bool
	mu        sync.RWMutex
}

// NewLDAPConnectionPool creates a new LDAP connection pool
func NewLDAPConnectionPool(config *LDAPConfig, logger *slog.Logger) *LDAPConnectionPool {
	if config.MaxConnections == 0 {
		config.MaxConnections = 10
	}
	
	minConnections := config.MaxConnections / 4
	if minConnections < 1 {
		minConnections = 1
	}
	
	pool := &LDAPConnectionPool{
		config:         config,
		logger:         logger.With("component", "ldap_pool"),
		connections:    make(chan *ldap.Conn, config.MaxConnections),
		maxConnections: config.MaxConnections,
		minConnections: minConnections,
		stats: &LDAPPoolStats{
			LastHealthCheck: time.Now(),
		},
	}
	
	// Create health checker
	pool.healthChecker = NewLDAPHealthChecker(config, logger)
	
	// Initialize minimum connections
	pool.initializeConnections()
	
	// Start background maintenance
	go pool.maintainConnections()
	
	return pool
}

// GetConnection retrieves a connection from the pool
func (p *LDAPConnectionPool) GetConnection(ctx context.Context) (*ldap.Conn, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("connection pool is closed")
	}
	p.mu.RUnlock()
	
	// Try to get existing connection from pool
	select {
	case conn := <-p.connections:
		if p.validateConnection(conn) {
			p.stats.mu.Lock()
			p.stats.ConnectionsReused++
			p.stats.ActiveConnections++
			p.stats.mu.Unlock()
			
			return conn, nil
		}
		// Connection is invalid, close it and create new one
		conn.Close()
		p.stats.mu.Lock()
		p.stats.ConnectionsFailed++
		p.stats.mu.Unlock()
		
	case <-ctx.Done():
		return nil, ctx.Err()
		
	default:
		// No connections available, try to create new one
	}
	
	// Create new connection if under limit
	p.mu.Lock()
	canCreate := p.activeConnections < p.maxConnections
	if canCreate {
		p.activeConnections++
	}
	p.mu.Unlock()
	
	if !canCreate {
		// Wait for available connection
		select {
		case conn := <-p.connections:
			if p.validateConnection(conn) {
				p.stats.mu.Lock()
				p.stats.ConnectionsReused++
				p.stats.ActiveConnections++
				p.stats.mu.Unlock()
				return conn, nil
			}
			conn.Close()
			
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	// Create new connection
	conn, err := p.createConnection(ctx)
	if err != nil {
		p.mu.Lock()
		p.activeConnections--
		p.mu.Unlock()
		
		p.stats.mu.Lock()
		p.stats.ConnectionsFailed++
		p.stats.mu.Unlock()
		
		return nil, fmt.Errorf("failed to create LDAP connection: %w", err)
	}
	
	p.stats.mu.Lock()
	p.stats.ConnectionsCreated++
	p.stats.ActiveConnections++
	p.stats.TotalConnections++
	p.stats.mu.Unlock()
	
	return conn, nil
}

// ReturnConnection returns a connection to the pool
func (p *LDAPConnectionPool) ReturnConnection(conn *ldap.Conn) {
	if conn == nil {
		return
	}
	
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()
	
	if closed {
		conn.Close()
		return
	}
	
	// Validate connection before returning to pool
	if !p.validateConnection(conn) {
		conn.Close()
		p.mu.Lock()
		p.activeConnections--
		p.mu.Unlock()
		return
	}
	
	// Try to return to pool
	select {
	case p.connections <- conn:
		p.stats.mu.Lock()
		p.stats.ActiveConnections--
		p.stats.IdleConnections++
		p.stats.mu.Unlock()
		
	default:
		// Pool is full, close connection
		conn.Close()
		p.mu.Lock()
		p.activeConnections--
		p.mu.Unlock()
	}
}

// Close closes all connections and shuts down the pool
func (p *LDAPConnectionPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()
	
	// Stop health checker
	if p.healthChecker != nil {
		p.healthChecker.Stop()
	}
	
	// Close all connections
	close(p.connections)
	for conn := range p.connections {
		if conn != nil {
			conn.Close()
		}
	}
	
	p.logger.Info("LDAP connection pool closed")
	return nil
}

// GetStats returns connection pool statistics
func (p *LDAPConnectionPool) GetStats() *LDAPPoolStats {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	stats := &LDAPPoolStats{}
	*stats = *p.stats
	
	p.mu.RLock()
	stats.ActiveConnections = p.activeConnections
	stats.IdleConnections = len(p.connections)
	p.mu.RUnlock()
	
	return stats
}

// IsHealthy returns the current health status
func (p *LDAPConnectionPool) IsHealthy() bool {
	if p.healthChecker != nil {
		return p.healthChecker.IsHealthy()
	}
	return true
}

// TestConnection tests connectivity to LDAP server
func (p *LDAPConnectionPool) TestConnection(ctx context.Context) error {
	conn, err := p.GetConnection(ctx)
	if err != nil {
		return err
	}
	defer p.ReturnConnection(conn)
	
	// Perform basic search test
	searchRequest := ldap.NewSearchRequest(
		p.config.BaseDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1, 1, false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	)
	
	_, err = conn.Search(searchRequest)
	return err
}

// Private methods

func (p *LDAPConnectionPool) initializeConnections() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	for i := 0; i < p.minConnections; i++ {
		conn, err := p.createConnection(ctx)
		if err != nil {
			p.logger.Warn("Failed to create initial connection", "error", err, "attempt", i+1)
			continue
		}
		
		select {
		case p.connections <- conn:
			p.stats.mu.Lock()
			p.stats.ConnectionsCreated++
			p.stats.TotalConnections++
			p.stats.IdleConnections++
			p.stats.mu.Unlock()
			
		default:
			conn.Close()
		}
	}
	
	p.logger.Info("LDAP connection pool initialized",
		"min_connections", p.minConnections,
		"max_connections", p.maxConnections,
		"initial_connections", len(p.connections))
}

func (p *LDAPConnectionPool) createConnection(ctx context.Context) (*ldap.Conn, error) {
	address := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)
	
	var conn *ldap.Conn
	var err error
	
	// Create connection with timeout using correct LDAP v3 API
	if p.config.UseSSL {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: p.config.SkipVerify,
		}
		conn, err = ldap.DialURL(fmt.Sprintf("ldaps://%s", address), ldap.DialWithTLSConfig(tlsConfig))
	} else {
		conn, err = ldap.DialURL(fmt.Sprintf("ldap://%s", address))
		
		if err == nil && p.config.UseTLS {
			err = conn.StartTLS(&tls.Config{
				ServerName:         p.config.Host,
				InsecureSkipVerify: p.config.SkipVerify,
			})
		}
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to dial LDAP server: %w", err)
	}
	
	// Set connection timeout
	conn.SetTimeout(p.config.Timeout)
	
	// Bind with service account if configured
	if p.config.BindDN != "" && p.config.BindPassword != "" {
		if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to bind to LDAP server: %w", err)
		}
	}
	
	return conn, nil
}

func (p *LDAPConnectionPool) validateConnection(conn *ldap.Conn) bool {
	if conn == nil {
		return false
	}
	
	// Try a simple bind test
	if p.config.BindDN != "" && p.config.BindPassword != "" {
		err := conn.Bind(p.config.BindDN, p.config.BindPassword)
		return err == nil
	}
	
	return true
}

func (p *LDAPConnectionPool) maintainConnections() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		p.mu.RLock()
		if p.closed {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()
		
		p.cleanupConnections()
		p.ensureMinimumConnections()
		
		// Update health check
		if p.healthChecker != nil {
			p.healthChecker.RunHealthCheck()
		}
	}
}

func (p *LDAPConnectionPool) cleanupConnections() {
	var validConnections []*ldap.Conn
	
	// Drain current connections and validate them
	for {
		select {
		case conn := <-p.connections:
			if p.validateConnection(conn) {
				validConnections = append(validConnections, conn)
			} else {
				conn.Close()
				p.mu.Lock()
				p.activeConnections--
				p.mu.Unlock()
			}
		default:
			// No more connections to check
			goto returnConnections
		}
	}
	
returnConnections:
	// Return valid connections to pool
	for _, conn := range validConnections {
		select {
		case p.connections <- conn:
		default:
			// Pool full, close excess connection
			conn.Close()
			p.mu.Lock()
			p.activeConnections--
			p.mu.Unlock()
		}
	}
}

func (p *LDAPConnectionPool) ensureMinimumConnections() {
	currentIdle := len(p.connections)
	if currentIdle >= p.minConnections {
		return
	}
	
	needed := p.minConnections - currentIdle
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	for i := 0; i < needed; i++ {
		conn, err := p.createConnection(ctx)
		if err != nil {
			p.logger.Debug("Failed to create maintenance connection", "error", err)
			continue
		}
		
		select {
		case p.connections <- conn:
			p.stats.mu.Lock()
			p.stats.ConnectionsCreated++
			p.stats.TotalConnections++
			p.stats.mu.Unlock()
			
		default:
			conn.Close()
		}
	}
}

// NewLDAPHealthChecker creates a new LDAP health checker
func NewLDAPHealthChecker(config *LDAPConfig, logger *slog.Logger) *LDAPHealthChecker {
	return &LDAPHealthChecker{
		config:   config,
		logger:   logger.With("component", "ldap_health"),
		interval: 60 * time.Second,
		timeout:  10 * time.Second,
		healthy:  true, // Assume healthy initially
	}
}

// Start starts the health checker
func (h *LDAPHealthChecker) Start() {
	go h.healthCheckLoop()
}

// Stop stops the health checker
func (h *LDAPHealthChecker) Stop() {
	// Implementation would add a done channel to stop the loop
	// For now, this is a placeholder
}

// IsHealthy returns the current health status
func (h *LDAPHealthChecker) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.healthy
}

// RunHealthCheck performs a health check
func (h *LDAPHealthChecker) RunHealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()
	
	start := time.Now()
	healthy := h.performHealthCheck(ctx)
	duration := time.Since(start)
	
	h.mu.Lock()
	h.healthy = healthy
	h.lastCheck = time.Now()
	h.mu.Unlock()
	
	if healthy {
		h.logger.Debug("LDAP health check passed", "duration", duration)
	} else {
		h.logger.Warn("LDAP health check failed", "duration", duration)
	}
	
	return healthy
}

func (h *LDAPHealthChecker) healthCheckLoop() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	
	for range ticker.C {
		h.RunHealthCheck()
	}
}

func (h *LDAPHealthChecker) performHealthCheck(ctx context.Context) bool {
	// Create a temporary connection for health check
	address := fmt.Sprintf("%s:%d", h.config.Host, h.config.Port)
	
	var conn *ldap.Conn
	var err error
	
	if h.config.UseSSL {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: h.config.SkipVerify,
		}
		conn, err = ldap.DialURL(fmt.Sprintf("ldaps://%s", address), ldap.DialWithTLSConfig(tlsConfig))
	} else {
		conn, err = ldap.DialURL(fmt.Sprintf("ldap://%s", address))
		if err == nil && h.config.UseTLS {
			err = conn.StartTLS(&tls.Config{
				ServerName:         h.config.Host,
				InsecureSkipVerify: h.config.SkipVerify,
			})
		}
	}
	
	if err != nil {
		return false
	}
	defer conn.Close()
	
	// Test bind
	if h.config.BindDN != "" && h.config.BindPassword != "" {
		if err := conn.Bind(h.config.BindDN, h.config.BindPassword); err != nil {
			return false
		}
	}
	
	// Test basic search
	searchRequest := ldap.NewSearchRequest(
		h.config.BaseDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1, 1, false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	)
	
	_, err = conn.Search(searchRequest)
	return err == nil
}

// Enhanced LDAP Provider with connection pooling
type EnhancedLDAPProvider struct {
	*LDAPClient
	pool *LDAPConnectionPool
}

// NewEnhancedLDAPProvider creates a new LDAP provider with connection pooling
func NewEnhancedLDAPProvider(config *LDAPConfig, logger *slog.Logger) *EnhancedLDAPProvider {
	// Create base provider
	baseProvider := NewLDAPClient(config, logger)
	
	// Create connection pool
	pool := NewLDAPConnectionPool(config, logger)
	
	return &EnhancedLDAPProvider{
		LDAPClient: baseProvider,
		pool:         pool,
	}
}

// Override Authenticate to use connection pool
func (p *EnhancedLDAPProvider) Authenticate(ctx context.Context, username, password string) (*UserInfo, error) {
	conn, err := p.pool.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection from pool: %w", err)
	}
	defer p.pool.ReturnConnection(conn)
	
	// Use the pooled connection for authentication
	return p.authenticateWithConnection(ctx, conn, username, password)
}

// Override SearchUser to use connection pool
func (p *EnhancedLDAPProvider) SearchUser(ctx context.Context, username string) (*UserInfo, error) {
	conn, err := p.pool.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection from pool: %w", err)
	}
	defer p.pool.ReturnConnection(conn)
	
	// Use the pooled connection for search
	return p.searchUserWithConnection(ctx, conn, username)
}

// GetPoolStats returns connection pool statistics
func (p *EnhancedLDAPProvider) GetPoolStats() *LDAPPoolStats {
	return p.pool.GetStats()
}

// IsHealthy returns health status
func (p *EnhancedLDAPProvider) IsHealthy() bool {
	return p.pool.IsHealthy()
}

// Close closes the connection pool
func (p *EnhancedLDAPProvider) Close() error {
	return p.pool.Close()
}

// TestConnection tests connectivity using the pool
func (p *EnhancedLDAPProvider) TestConnection(ctx context.Context) error {
	return p.pool.TestConnection(ctx)
}

// Private helper methods that use specific connections

func (p *EnhancedLDAPProvider) authenticateWithConnection(ctx context.Context, conn *ldap.Conn, username, password string) (*UserInfo, error) {
	// Implementation would mirror the original Authenticate method but use the provided connection
	// This is a simplified version - the full implementation would use the same logic as the original
	
	// Search for user
	userDN, ldapUser, err := p.findUserWithConnection(conn, username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	
	// Try to bind with user credentials
	testConn, err := p.pool.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get test connection: %w", err)
	}
	defer p.pool.ReturnConnection(testConn)
	
	err = testConn.Bind(userDN, password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: invalid credentials")
	}
	
	// Get user groups
	groups, err := p.getUserGroupsWithConnection(ctx, conn, username)
	if err != nil {
		groups = []string{} // Continue without groups
	}
	
	// Get user roles from group mappings
	roles := p.mapGroupsToRoles(groups)
	
	// Convert to standard UserInfo
	userInfo := &UserInfo{
		Subject:       ldapUser.Username,
		Email:         ldapUser.Email,
		EmailVerified: ldapUser.Email != "",
		Name:          ldapUser.DisplayName,
		GivenName:     ldapUser.FirstName,
		FamilyName:    ldapUser.LastName,
		Username:      ldapUser.Username,
		Groups:        groups,
		Roles:         roles,
		Provider:      "ldap",
		ProviderID:    userDN,
		Attributes: map[string]interface{}{
			"ldap_dn":     ldapUser.DN,
			"title":       ldapUser.Title,
			"department":  ldapUser.Department,
			"phone":       ldapUser.Phone,
			"manager":     ldapUser.Manager,
			"active":      ldapUser.Active,
		},
	}
	
	return userInfo, nil
}

func (p *EnhancedLDAPProvider) searchUserWithConnection(ctx context.Context, conn *ldap.Conn, username string) (*UserInfo, error) {
	// Similar to the original SearchUser but using provided connection
	userDN, ldapUser, err := p.findUserWithConnection(conn, username)
	if err != nil {
		return nil, err
	}
	
	groups, err := p.getUserGroupsWithConnection(ctx, conn, username)
	if err != nil {
		groups = []string{}
	}
	
	roles := p.mapGroupsToRoles(groups)
	
	userInfo := &UserInfo{
		Subject:       ldapUser.Username,
		Email:         ldapUser.Email,
		EmailVerified: ldapUser.Email != "",
		Name:          ldapUser.DisplayName,
		GivenName:     ldapUser.FirstName,
		FamilyName:    ldapUser.LastName,
		Username:      ldapUser.Username,
		Groups:        groups,
		Roles:         roles,
		Provider:      "ldap",
		ProviderID:    userDN,
		Attributes: map[string]interface{}{
			"ldap_dn":     ldapUser.DN,
			"title":       ldapUser.Title,
			"department":  ldapUser.Department,
			"phone":       ldapUser.Phone,
			"manager":     ldapUser.Manager,
			"active":      ldapUser.Active,
		},
	}
	
	return userInfo, nil
}

func (p *EnhancedLDAPProvider) findUserWithConnection(conn *ldap.Conn, username string) (string, *LDAPUserInfo, error) {
	// Implementation mirrors the original findUser method but uses provided connection
	// This is a placeholder - the full implementation would use the same logic
	return "", nil, fmt.Errorf("not implemented - would use connection to find user")
}

func (p *EnhancedLDAPProvider) getUserGroupsWithConnection(ctx context.Context, conn *ldap.Conn, username string) ([]string, error) {
	// Implementation mirrors the original GetUserGroups method but uses provided connection
	// This is a placeholder - the full implementation would use the same logic
	return []string{}, nil
}
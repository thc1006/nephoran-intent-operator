//go:build go1.24

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// BenchmarkAuthSystemSuite provides comprehensive authentication and authorization benchmarks using Go 1.24+ features
func BenchmarkAuthSystemSuite(b *testing.B) {
	ctx := context.Background()

	// Setup enhanced auth system for benchmarking
	authSystem := setupBenchmarkAuthSystem()
	defer authSystem.Cleanup()

	b.Run("JWTValidation", func(b *testing.B) {
		benchmarkJWTValidation(b, ctx, authSystem)
	})

	b.Run("RBACAuthorization", func(b *testing.B) {
		benchmarkRBACAuthorization(b, ctx, authSystem)
	})

	b.Run("LDAPAuthentication", func(b *testing.B) {
		benchmarkLDAPAuthentication(b, ctx, authSystem)
	})

	b.Run("OAuth2TokenExchange", func(b *testing.B) {
		benchmarkOAuth2TokenExchange(b, ctx, authSystem)
	})

	b.Run("SessionManagement", func(b *testing.B) {
		benchmarkSessionManagement(b, ctx, authSystem)
	})

	b.Run("ConcurrentAuthentication", func(b *testing.B) {
		benchmarkConcurrentAuthentication(b, ctx, authSystem)
	})

	b.Run("TokenCaching", func(b *testing.B) {
		benchmarkTokenCaching(b, ctx, authSystem)
	})

	b.Run("PermissionMatrix", func(b *testing.B) {
		benchmarkPermissionMatrix(b, ctx, authSystem)
	})
}

// benchmarkJWTValidation tests JWT token validation performance with different token complexities
func benchmarkJWTValidation(b *testing.B, ctx context.Context, authSystem *EnhancedAuthSystem) {
	jwtScenarios := []struct {
		name        string
		algorithm   string
		keySize     int
		claimsCount int
		tokenExpiry time.Duration
	}{
		{"RS256_Small", "RS256", 2048, 5, time.Hour},
		{"RS256_Medium", "RS256", 2048, 15, time.Hour},
		{"RS256_Large", "RS256", 2048, 30, time.Hour},
		{"RS512_Large", "RS512", 4096, 30, time.Hour},
		{"ExpiredToken", "RS256", 2048, 10, -time.Hour}, // Expired token
	}

	for _, scenario := range jwtScenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Generate test tokens with specified parameters
			tokens := generateJWTTokens(scenario.algorithm, scenario.keySize,
				scenario.claimsCount, scenario.tokenExpiry, 1000) // Pre-generate 1000 tokens

			var validationLatency int64
			var validTokens, invalidTokens, expiredTokens int64
			var parseErrors, signatureErrors int64

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				token := tokens[i%len(tokens)]

				valStart := time.Now()
				result, err := authSystem.ValidateJWTToken(ctx, token)
				valLatency := time.Since(valStart)

				atomic.AddInt64(&validationLatency, valLatency.Nanoseconds())

				if err != nil {
					switch {
					case isTokenExpiredError(err):
						atomic.AddInt64(&expiredTokens, 1)
					case isSignatureError(err):
						atomic.AddInt64(&signatureErrors, 1)
					default:
						atomic.AddInt64(&parseErrors, 1)
					}
				} else {
					if result.Valid {
						atomic.AddInt64(&validTokens, 1)
					} else {
						atomic.AddInt64(&invalidTokens, 1)
					}
				}
			}

			// Calculate JWT validation metrics
			avgLatency := time.Duration(validationLatency / int64(b.N))
			validationRate := float64(b.N) / b.Elapsed().Seconds()
			validityRate := float64(validTokens) / float64(b.N) * 100
			expiryRate := float64(expiredTokens) / float64(b.N) * 100
			signatureErrorRate := float64(signatureErrors) / float64(b.N) * 100

			b.ReportMetric(float64(avgLatency.Microseconds()), "avg_validation_latency_us")
			b.ReportMetric(validationRate, "validations_per_sec")
			b.ReportMetric(validityRate, "valid_token_rate_percent")
			b.ReportMetric(expiryRate, "expired_token_rate_percent")
			b.ReportMetric(signatureErrorRate, "signature_error_rate_percent")
			b.ReportMetric(float64(scenario.claimsCount), "claims_count")
			b.ReportMetric(float64(scenario.keySize), "key_size_bits")
		})
	}
}

// benchmarkRBACAuthorization tests role-based access control performance
func benchmarkRBACAuthorization(b *testing.B, ctx context.Context, authSystem *EnhancedAuthSystem) {
	rbacScenarios := []struct {
		name            string
		roleCount       int
		permissionCount int
		resourceTypes   int
		userGroups      int
		hierarchyDepth  int
	}{
		{"Simple_RBAC", 5, 10, 3, 2, 1},
		{"Medium_RBAC", 20, 50, 8, 5, 2},
		{"Complex_RBAC", 50, 200, 15, 10, 3},
		{"Enterprise_RBAC", 100, 500, 25, 20, 4},
	}

	for _, scenario := range rbacScenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Setup RBAC configuration
			rbacConfig := setupRBACConfiguration(scenario.roleCount, scenario.permissionCount,
				scenario.resourceTypes, scenario.userGroups, scenario.hierarchyDepth)

			authSystem.ConfigureRBAC(rbacConfig)

			// Generate test authorization requests
			authRequests := generateAuthorizationRequests(scenario.resourceTypes, 100)
			testUser := generateTestUser("test-user", scenario.userGroups/2) // User in half the groups

			var authzLatency int64
			var authorized, denied int64
			var roleEvaluations, permissionChecks int64
			var cacheHits, cacheMisses int64

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				request := authRequests[i%len(authRequests)]

				authzStart := time.Now()
				result, err := authSystem.AuthorizeRequest(ctx, testUser, request)
				authzLatency := time.Since(authzStart)

				atomic.AddInt64(&authzLatency, authzLatency.Nanoseconds())

				if err != nil {
					b.Errorf("Authorization failed: %v", err)
				} else {
					if result.Authorized {
						atomic.AddInt64(&authorized, 1)
					} else {
						atomic.AddInt64(&denied, 1)
					}

					atomic.AddInt64(&roleEvaluations, int64(result.RolesEvaluated))
					atomic.AddInt64(&permissionChecks, int64(result.PermissionsChecked))

					if result.CacheHit {
						atomic.AddInt64(&cacheHits, 1)
					} else {
						atomic.AddInt64(&cacheMisses, 1)
					}
				}
			}

			// Calculate RBAC authorization metrics
			avgAuthzLatency := time.Duration(authzLatency / int64(b.N))
			authzRate := float64(b.N) / b.Elapsed().Seconds()
			authorizationRate := float64(authorized) / float64(b.N) * 100
			avgRoleEvals := float64(roleEvaluations) / float64(b.N)
			avgPermissionChecks := float64(permissionChecks) / float64(b.N)
			cacheHitRate := float64(cacheHits) / float64(cacheHits+cacheMisses) * 100

			b.ReportMetric(float64(avgAuthzLatency.Microseconds()), "avg_authz_latency_us")
			b.ReportMetric(authzRate, "authorizations_per_sec")
			b.ReportMetric(authorizationRate, "authorization_rate_percent")
			b.ReportMetric(avgRoleEvals, "avg_roles_evaluated")
			b.ReportMetric(avgPermissionChecks, "avg_permissions_checked")
			b.ReportMetric(cacheHitRate, "cache_hit_rate_percent")
			b.ReportMetric(float64(scenario.roleCount), "role_count")
			b.ReportMetric(float64(scenario.permissionCount), "permission_count")
		})
	}
}

// benchmarkLDAPAuthentication tests LDAP authentication performance
func benchmarkLDAPAuthentication(b *testing.B, ctx context.Context, authSystem *EnhancedAuthSystem) {
	ldapScenarios := []struct {
		name              string
		userPoolSize      int
		groupDepth        int
		attributeCount    int
		useConnectionPool bool
	}{
		{"Small_Pool", 100, 2, 5, false},
		{"Medium_Pool", 1000, 3, 10, true},
		{"Large_Pool", 10000, 4, 15, true},
		{"Connection_Pool", 1000, 3, 10, true},
	}

	for _, scenario := range ldapScenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Configure LDAP settings
			ldapConfig := LDAPConfig{
				Host:               "localhost:389",
				BaseDN:             "dc=example,dc=com",
				UserSearchBase:     "ou=users,dc=example,dc=com",
				GroupSearchBase:    "ou=groups,dc=example,dc=com",
				ConnectionPoolSize: 10,
				UseConnectionPool:  scenario.useConnectionPool,
			}

			authSystem.ConfigureLDAP(ldapConfig)

			// Generate test users
			testUsers := generateLDAPTestUsers(scenario.userPoolSize, scenario.groupDepth)

			var authLatency, bindLatency, searchLatency int64
			var successfulAuths, failedAuths int64
			var connectionPoolHits int64
			var groupLookups int64

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				user := testUsers[i%len(testUsers)]

				authStart := time.Now()
				result, err := authSystem.AuthenticateLDAP(ctx, user.Username, user.Password)
				authLatency := time.Since(authStart)

				atomic.AddInt64(&authLatency, authLatency.Nanoseconds())

				if err != nil {
					atomic.AddInt64(&failedAuths, 1)
				} else {
					atomic.AddInt64(&successfulAuths, 1)
					atomic.AddInt64(&bindLatency, int64(result.BindTime.Nanoseconds()))
					atomic.AddInt64(&searchLatency, int64(result.SearchTime.Nanoseconds()))
					atomic.AddInt64(&groupLookups, int64(result.GroupsFound))

					if result.UsedConnectionPool {
						atomic.AddInt64(&connectionPoolHits, 1)
					}
				}
			}

			// Calculate LDAP authentication metrics
			avgAuthLatency := time.Duration(authLatency / int64(b.N))
			avgBindLatency := time.Duration(bindLatency / int64(b.N))
			avgSearchLatency := time.Duration(searchLatency / int64(b.N))
			authRate := float64(b.N) / b.Elapsed().Seconds()
			successRate := float64(successfulAuths) / float64(b.N) * 100
			avgGroupLookups := float64(groupLookups) / float64(b.N)
			poolUtilization := float64(connectionPoolHits) / float64(b.N) * 100

			b.ReportMetric(float64(avgAuthLatency.Milliseconds()), "avg_auth_latency_ms")
			b.ReportMetric(float64(avgBindLatency.Milliseconds()), "avg_bind_latency_ms")
			b.ReportMetric(float64(avgSearchLatency.Milliseconds()), "avg_search_latency_ms")
			b.ReportMetric(authRate, "authentications_per_sec")
			b.ReportMetric(successRate, "success_rate_percent")
			b.ReportMetric(avgGroupLookups, "avg_groups_per_user")
			b.ReportMetric(poolUtilization, "connection_pool_utilization_percent")
			b.ReportMetric(float64(scenario.userPoolSize), "user_pool_size")
		})
	}
}

// benchmarkOAuth2TokenExchange tests OAuth2 token exchange performance
func benchmarkOAuth2TokenExchange(b *testing.B, ctx context.Context, authSystem *EnhancedAuthSystem) {
	oauth2Scenarios := []struct {
		name       string
		provider   string
		tokenType  string
		scopeCount int
		audience   string
	}{
		{"GitHub_AccessToken", "github", "access_token", 3, "nephoran-api"},
		{"Google_IDToken", "google", "id_token", 5, "nephoran-api"},
		{"AzureAD_AccessToken", "azuread", "access_token", 8, "nephoran-api"},
		{"Custom_RefreshToken", "custom", "refresh_token", 4, "nephoran-api"},
	}

	for _, scenario := range oauth2Scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Configure OAuth2 provider
			oauth2Config := OAuth2Config{
				Provider:     scenario.provider,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Scopes:       generateScopes(scenario.scopeCount),
				Audience:     scenario.audience,
			}

			authSystem.ConfigureOAuth2(oauth2Config)

			// Generate test OAuth2 tokens
			oauth2Tokens := generateOAuth2Tokens(scenario.provider, scenario.tokenType, 100)

			var exchangeLatency, validationLatency int64
			var successfulExchanges, failedExchanges int64
			var tokenRefreshes int64
			var scopeValidations int64

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				token := oauth2Tokens[i%len(oauth2Tokens)]

				exchangeStart := time.Now()
				result, err := authSystem.ExchangeOAuth2Token(ctx, token, oauth2Config)
				exchangeLatency := time.Since(exchangeStart)

				atomic.AddInt64(&exchangeLatency, exchangeLatency.Nanoseconds())

				if err != nil {
					atomic.AddInt64(&failedExchanges, 1)
				} else {
					atomic.AddInt64(&successfulExchanges, 1)
					atomic.AddInt64(&validationLatency, int64(result.ValidationTime.Nanoseconds()))
					atomic.AddInt64(&scopeValidations, int64(result.ScopesValidated))

					if result.TokenRefreshed {
						atomic.AddInt64(&tokenRefreshes, 1)
					}
				}
			}

			// Calculate OAuth2 exchange metrics
			avgExchangeLatency := time.Duration(exchangeLatency / int64(b.N))
			avgValidationLatency := time.Duration(validationLatency / int64(b.N))
			exchangeRate := float64(b.N) / b.Elapsed().Seconds()
			successRate := float64(successfulExchanges) / float64(b.N) * 100
			refreshRate := float64(tokenRefreshes) / float64(b.N) * 100
			avgScopeValidations := float64(scopeValidations) / float64(b.N)

			b.ReportMetric(float64(avgExchangeLatency.Milliseconds()), "avg_exchange_latency_ms")
			b.ReportMetric(float64(avgValidationLatency.Milliseconds()), "avg_validation_latency_ms")
			b.ReportMetric(exchangeRate, "exchanges_per_sec")
			b.ReportMetric(successRate, "success_rate_percent")
			b.ReportMetric(refreshRate, "token_refresh_rate_percent")
			b.ReportMetric(avgScopeValidations, "avg_scopes_validated")
			b.ReportMetric(float64(scenario.scopeCount), "scope_count")
		})
	}
}

// benchmarkSessionManagement tests session lifecycle management performance
func benchmarkSessionManagement(b *testing.B, ctx context.Context, authSystem *EnhancedAuthSystem) {
	sessionScenarios := []struct {
		name           string
		operation      string
		concurrency    int
		sessionTTL     time.Duration
		storageBackend string
	}{
		{"CreateSession", "create", 1, time.Hour, "memory"},
		{"ValidateSession", "validate", 1, time.Hour, "memory"},
		{"RefreshSession", "refresh", 1, time.Hour, "memory"},
		{"DeleteSession", "delete", 1, time.Hour, "memory"},
		{"ConcurrentCreate", "create", 10, time.Hour, "redis"},
		{"ConcurrentValidate", "validate", 20, time.Hour, "redis"},
	}

	for _, scenario := range sessionScenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Configure session management
			sessionConfig := SessionConfig{
				TTL:             scenario.sessionTTL,
				StorageBackend:  scenario.storageBackend,
				CleanupInterval: time.Minute * 5,
			}

			authSystem.ConfigureSessionManagement(sessionConfig)

			// Pre-create sessions for validation/refresh/delete operations
			var existingSessions []string
			if scenario.operation != "create" {
				for i := 0; i < 1000; i++ {
					sessionID, _ := authSystem.CreateSession(ctx, fmt.Sprintf("user-%d", i))
					existingSessions = append(existingSessions, sessionID)
				}
			}

			var operationLatency int64
			var successfulOps, failedOps int64
			var cacheHits, cacheMisses int64

			b.ResetTimer()
			b.ReportAllocs()

			semaphore := make(chan struct{}, scenario.concurrency)

			for i := 0; i < b.N; i++ {
				semaphore <- struct{}{}

				go func(iteration int) {
					defer func() { <-semaphore }()

					opStart := time.Now()
					var err error
					var cacheHit bool

					switch scenario.operation {
					case "create":
						_, err = authSystem.CreateSession(ctx, fmt.Sprintf("user-%d", iteration))
					case "validate":
						sessionID := existingSessions[iteration%len(existingSessions)]
						var result *SessionValidationResult
						result, err = authSystem.ValidateSession(ctx, sessionID)
						if result != nil {
							cacheHit = result.FromCache
						}
					case "refresh":
						sessionID := existingSessions[iteration%len(existingSessions)]
						err = authSystem.RefreshSession(ctx, sessionID)
					case "delete":
						sessionID := existingSessions[iteration%len(existingSessions)]
						err = authSystem.DeleteSession(ctx, sessionID)
					}

					opLatency := time.Since(opStart)
					atomic.AddInt64(&operationLatency, opLatency.Nanoseconds())

					if err != nil {
						atomic.AddInt64(&failedOps, 1)
					} else {
						atomic.AddInt64(&successfulOps, 1)

						if cacheHit {
							atomic.AddInt64(&cacheHits, 1)
						} else {
							atomic.AddInt64(&cacheMisses, 1)
						}
					}
				}(i)
			}

			// Wait for all operations to complete
			for i := 0; i < scenario.concurrency; i++ {
				semaphore <- struct{}{}
			}

			// Calculate session management metrics
			avgOpLatency := time.Duration(operationLatency / int64(b.N))
			opRate := float64(b.N) / b.Elapsed().Seconds()
			successRate := float64(successfulOps) / float64(b.N) * 100
			cacheHitRate := float64(cacheHits) / float64(cacheHits+cacheMisses) * 100

			b.ReportMetric(float64(avgOpLatency.Microseconds()), "avg_operation_latency_us")
			b.ReportMetric(opRate, "operations_per_sec")
			b.ReportMetric(successRate, "success_rate_percent")
			b.ReportMetric(cacheHitRate, "cache_hit_rate_percent")
			b.ReportMetric(float64(scenario.concurrency), "concurrency_level")
		})
	}
}

// benchmarkConcurrentAuthentication tests authentication system under concurrent load
func benchmarkConcurrentAuthentication(b *testing.B, ctx context.Context, authSystem *EnhancedAuthSystem) {
	concurrencyLevels := []int{1, 5, 10, 25, 50, 100}

	// Prepare test data
	testUsers := generateTestUsers(1000)
	testTokens := generateJWTTokens("RS256", 2048, 10, time.Hour, 1000)

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrent-%d", concurrency), func(b *testing.B) {
			var totalLatency int64
			var jwtValidations, ldapAuths, oauth2Exchanges int64
			var successCount, errorCount int64

			// Enhanced memory tracking for concurrent operations
			var startMemStats, peakMemStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startMemStats)
			peakMemory := int64(startMemStats.Alloc)

			b.ResetTimer()
			b.ReportAllocs()

			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				localIterations := 0
				for pb.Next() {
					// Mix different authentication methods
					authType := localIterations % 3

					start := time.Now()
					var err error

					switch authType {
					case 0: // JWT validation
						token := testTokens[localIterations%len(testTokens)]
						_, err = authSystem.ValidateJWTToken(ctx, token)
						atomic.AddInt64(&jwtValidations, 1)
					case 1: // LDAP authentication
						user := testUsers[localIterations%len(testUsers)]
						_, err = authSystem.AuthenticateLDAP(ctx, user.Username, user.Password)
						atomic.AddInt64(&ldapAuths, 1)
					case 2: // OAuth2 exchange
						oauth2Token := OAuth2Token{
							AccessToken: fmt.Sprintf("oauth2-token-%d", localIterations),
							Provider:    "github",
						}
						_, err = authSystem.ExchangeOAuth2Token(ctx, oauth2Token, OAuth2Config{
							Provider: "github",
							ClientID: "test-client",
						})
						atomic.AddInt64(&oauth2Exchanges, 1)
					}

					latency := time.Since(start)
					atomic.AddInt64(&totalLatency, latency.Nanoseconds())

					if err != nil {
						atomic.AddInt64(&errorCount, 1)
					} else {
						atomic.AddInt64(&successCount, 1)
					}

					// Track peak memory usage
					var currentMemStats runtime.MemStats
					runtime.ReadMemStats(&currentMemStats)
					currentAlloc := int64(currentMemStats.Alloc)
					if currentAlloc > peakMemory {
						peakMemory = currentAlloc
						peakMemStats = currentMemStats
					}

					localIterations++
				}
			})

			// Calculate concurrent authentication metrics
			totalRequests := int64(b.N)
			avgLatency := time.Duration(totalLatency / totalRequests)
			throughput := float64(totalRequests) / b.Elapsed().Seconds()
			successRate := float64(successCount) / float64(totalRequests) * 100
			memoryGrowth := float64(peakMemory-int64(startMemStats.Alloc)) / 1024 / 1024 // MB

			jwtValidationRate := float64(jwtValidations) / float64(totalRequests) * 100
			ldapAuthRate := float64(ldapAuths) / float64(totalRequests) * 100
			oauth2ExchangeRate := float64(oauth2Exchanges) / float64(totalRequests) * 100

			b.ReportMetric(float64(avgLatency.Milliseconds()), "avg_latency_ms")
			b.ReportMetric(throughput, "requests_per_sec")
			b.ReportMetric(successRate, "success_rate_percent")
			b.ReportMetric(memoryGrowth, "memory_growth_mb")
			b.ReportMetric(jwtValidationRate, "jwt_validation_rate_percent")
			b.ReportMetric(ldapAuthRate, "ldap_auth_rate_percent")
			b.ReportMetric(oauth2ExchangeRate, "oauth2_exchange_rate_percent")
			b.ReportMetric(float64(concurrency), "concurrency_level")
		})
	}
}

// benchmarkTokenCaching tests token validation caching effectiveness
func benchmarkTokenCaching(b *testing.B, ctx context.Context, authSystem *EnhancedAuthSystem) {
	cacheScenarios := []struct {
		name           string
		cacheSize      int
		cacheTTL       time.Duration
		uniqueTokens   int
		requestPattern string
	}{
		{"SmallCache_HighHit", 100, time.Minute * 5, 50, "repeated"},
		{"MediumCache_MediumHit", 500, time.Minute * 10, 200, "mixed"},
		{"LargeCache_LowHit", 1000, time.Minute * 15, 800, "unique"},
		{"TTL_Expiration", 200, time.Second * 30, 100, "repeated"},
	}

	for _, scenario := range cacheScenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Configure token cache
			cacheConfig := TokenCacheConfig{
				MaxSize: scenario.cacheSize,
				TTL:     scenario.cacheTTL,
			}

			authSystem.ConfigureTokenCache(cacheConfig)

			// Generate token set based on scenario
			tokens := generateJWTTokens("RS256", 2048, 10, time.Hour, scenario.uniqueTokens)

			var validationLatency int64
			var cacheHits, cacheMisses int64
			var validTokens, invalidTokens int64

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var token string

				// Select token based on request pattern
				switch scenario.requestPattern {
				case "repeated":
					// Favor first 20% of tokens to increase cache hits
					token = tokens[i%(scenario.uniqueTokens/5)]
				case "mixed":
					// Mix of repeated and unique tokens
					if i%3 == 0 {
						token = tokens[i%(scenario.uniqueTokens/3)]
					} else {
						token = tokens[i%scenario.uniqueTokens]
					}
				case "unique":
					// Mostly unique tokens to decrease cache hits
					token = tokens[i%scenario.uniqueTokens]
				}

				valStart := time.Now()
				result, err := authSystem.ValidateJWTTokenCached(ctx, token)
				valLatency := time.Since(valStart)

				atomic.AddInt64(&validationLatency, valLatency.Nanoseconds())

				if err != nil {
					atomic.AddInt64(&invalidTokens, 1)
				} else {
					atomic.AddInt64(&validTokens, 1)

					if result.FromCache {
						atomic.AddInt64(&cacheHits, 1)
					} else {
						atomic.AddInt64(&cacheMisses, 1)
					}
				}
			}

			// Calculate token caching metrics
			avgLatency := time.Duration(validationLatency / int64(b.N))
			validationRate := float64(b.N) / b.Elapsed().Seconds()
			cacheHitRate := float64(cacheHits) / float64(cacheHits+cacheMisses) * 100
			cacheEffectiveness := cacheHitRate / 100.0 * validationRate // Effective throughput from cache

			b.ReportMetric(float64(avgLatency.Microseconds()), "avg_validation_latency_us")
			b.ReportMetric(validationRate, "validations_per_sec")
			b.ReportMetric(cacheHitRate, "cache_hit_rate_percent")
			b.ReportMetric(cacheEffectiveness, "cache_effectiveness_score")
			b.ReportMetric(float64(scenario.cacheSize), "cache_size")
			b.ReportMetric(float64(scenario.uniqueTokens), "unique_tokens")
		})
	}
}

// benchmarkPermissionMatrix tests large-scale permission evaluation
func benchmarkPermissionMatrix(b *testing.B, ctx context.Context, authSystem *EnhancedAuthSystem) {
	matrixScenarios := []struct {
		name            string
		userCount       int
		roleCount       int
		resourceCount   int
		permissionTypes int
		matrixDensity   float64 // Percentage of user-resource-permission combinations that are valid
	}{
		{"Small_Matrix", 100, 10, 50, 5, 0.1},
		{"Medium_Matrix", 1000, 50, 200, 10, 0.05},
		{"Large_Matrix", 5000, 200, 1000, 20, 0.02},
		{"Sparse_Matrix", 10000, 100, 500, 15, 0.01},
	}

	for _, scenario := range matrixScenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Generate permission matrix
			permissionMatrix := generatePermissionMatrix(scenario.userCount, scenario.roleCount,
				scenario.resourceCount, scenario.permissionTypes, scenario.matrixDensity)

			authSystem.LoadPermissionMatrix(permissionMatrix)

			// Generate test permission checks
			permissionChecks := generatePermissionChecks(scenario.userCount, scenario.resourceCount,
				scenario.permissionTypes, 1000)

			var evaluationLatency int64
			var matrixLookups int64
			var permissionGrants, permissionDenials int64
			var cacheUtilization int64

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				check := permissionChecks[i%len(permissionChecks)]

				evalStart := time.Now()
				result, err := authSystem.EvaluatePermission(ctx, check)
				evalLatency := time.Since(evalStart)

				atomic.AddInt64(&evaluationLatency, evalLatency.Nanoseconds())

				if err != nil {
					b.Errorf("Permission evaluation failed: %v", err)
				} else {
					atomic.AddInt64(&matrixLookups, int64(result.MatrixLookups))

					if result.Granted {
						atomic.AddInt64(&permissionGrants, 1)
					} else {
						atomic.AddInt64(&permissionDenials, 1)
					}

					if result.UsedCache {
						atomic.AddInt64(&cacheUtilization, 1)
					}
				}
			}

			// Calculate permission matrix metrics
			avgEvalLatency := time.Duration(evaluationLatency / int64(b.N))
			evaluationRate := float64(b.N) / b.Elapsed().Seconds()
			grantRate := float64(permissionGrants) / float64(b.N) * 100
			avgMatrixLookups := float64(matrixLookups) / float64(b.N)
			cacheUtilizationRate := float64(cacheUtilization) / float64(b.N) * 100

			// Calculate matrix efficiency metrics
			expectedGrants := scenario.matrixDensity * 100
			matrixAccuracy := 100 - abs(grantRate-expectedGrants) // How close we are to expected grants

			b.ReportMetric(float64(avgEvalLatency.Microseconds()), "avg_evaluation_latency_us")
			b.ReportMetric(evaluationRate, "evaluations_per_sec")
			b.ReportMetric(grantRate, "permission_grant_rate_percent")
			b.ReportMetric(avgMatrixLookups, "avg_matrix_lookups")
			b.ReportMetric(cacheUtilizationRate, "cache_utilization_percent")
			b.ReportMetric(matrixAccuracy, "matrix_accuracy_percent")
			b.ReportMetric(float64(scenario.userCount), "user_count")
			b.ReportMetric(float64(scenario.resourceCount), "resource_count")
		})
	}
}

// Helper functions and data generators

func generateJWTTokens(algorithm string, keySize int, claimsCount int, expiry time.Duration, count int) []string {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate RSA key: %v", err))
	}

	tokens := make([]string, count)

	for i := 0; i < count; i++ {
		// Create claims
		claims := jwt.MapClaims{
			"sub": fmt.Sprintf("user-%d", i),
			"iss": "nephoran-test",
			"aud": "nephoran-api",
			"iat": time.Now().Unix(),
		}

		if expiry > 0 {
			claims["exp"] = time.Now().Add(expiry).Unix()
		} else {
			claims["exp"] = time.Now().Add(expiry).Unix() // Negative expiry for expired tokens
		}

		// Add additional claims based on claimsCount
		for j := 0; j < claimsCount; j++ {
			claims[fmt.Sprintf("claim_%d", j)] = fmt.Sprintf("value_%d_%d", i, j)
		}

		// Create and sign token
		token := jwt.NewWithClaims(jwt.GetSigningMethod(algorithm), claims)
		signedToken, err := token.SignedString(privateKey)
		if err != nil {
			panic(fmt.Sprintf("Failed to sign token: %v", err))
		}

		tokens[i] = signedToken
	}

	return tokens
}

func setupRBACConfiguration(roleCount, permissionCount, resourceTypes, userGroups, hierarchyDepth int) RBACConfig {
	return RBACConfig{
		Roles:          generateRoles(roleCount, permissionCount),
		Permissions:    generatePermissions(permissionCount, resourceTypes),
		UserGroups:     generateUserGroups(userGroups),
		HierarchyDepth: hierarchyDepth,
	}
}

func generateRoles(count, permissionCount int) []Role {
	roles := make([]Role, count)

	for i := range roles {
		roles[i] = Role{
			Name:        fmt.Sprintf("role-%d", i),
			Permissions: generateRolePermissions(i, permissionCount),
		}
	}

	return roles
}

func generateRolePermissions(roleIndex, maxPermissions int) []string {
	// Each role gets a subset of permissions
	permissionCount := (roleIndex % maxPermissions) + 1
	permissions := make([]string, permissionCount)

	for i := range permissions {
		permissions[i] = fmt.Sprintf("permission-%d", (roleIndex+i)%maxPermissions)
	}

	return permissions
}

func generatePermissions(count, resourceTypes int) []Permission {
	permissions := make([]Permission, count)

	actions := []string{"read", "write", "delete", "create", "update"}

	for i := range permissions {
		permissions[i] = Permission{
			Name:     fmt.Sprintf("permission-%d", i),
			Resource: fmt.Sprintf("resource-type-%d", i%resourceTypes),
			Action:   actions[i%len(actions)],
		}
	}

	return permissions
}

func generateUserGroups(count int) []UserGroup {
	groups := make([]UserGroup, count)

	for i := range groups {
		groups[i] = UserGroup{
			Name:  fmt.Sprintf("group-%d", i),
			Users: []string{fmt.Sprintf("user-%d", i), fmt.Sprintf("user-%d", i+count)},
		}
	}

	return groups
}

func generateAuthorizationRequests(resourceTypes, count int) []AuthorizationRequest {
	requests := make([]AuthorizationRequest, count)
	actions := []string{"read", "write", "delete", "create", "update"}

	for i := range requests {
		requests[i] = AuthorizationRequest{
			Resource: fmt.Sprintf("resource-%d", i%resourceTypes),
			Action:   actions[i%len(actions)],
			Context:  map[string]interface{}{"tenant": fmt.Sprintf("tenant-%d", i%10)},
		}
	}

	return requests
}

func generateTestUser(username string, groupCount int) User {
	groups := make([]string, groupCount)
	for i := range groups {
		groups[i] = fmt.Sprintf("group-%d", i)
	}

	return User{
		Username: username,
		Groups:   groups,
		Attributes: map[string]interface{}{
			"department": "engineering",
			"level":      "senior",
		},
	}
}

func generateLDAPTestUsers(count, groupDepth int) []LDAPUser {
	users := make([]LDAPUser, count)

	for i := range users {
		users[i] = LDAPUser{
			Username: fmt.Sprintf("user-%d", i),
			Password: fmt.Sprintf("password-%d", i),
			DN:       fmt.Sprintf("uid=user-%d,ou=users,dc=example,dc=com", i),
			Groups:   generateUserLDAPGroups(i, groupDepth),
		}
	}

	return users
}

func generateUserLDAPGroups(userIndex, depth int) []string {
	groupCount := (userIndex % depth) + 1
	groups := make([]string, groupCount)

	for i := range groups {
		groups[i] = fmt.Sprintf("cn=group-%d,ou=groups,dc=example,dc=com", (userIndex+i)%10)
	}

	return groups
}

func generateScopes(count int) []string {
	scopes := make([]string, count)
	baseSopes := []string{"read", "write", "admin", "user", "api", "openid", "profile", "email"}

	for i := range scopes {
		scopes[i] = baseSopes[i%len(baseSopes)]
	}

	return scopes
}

func generateOAuth2Tokens(provider, tokenType string, count int) []OAuth2Token {
	tokens := make([]OAuth2Token, count)

	for i := range tokens {
		tokens[i] = OAuth2Token{
			AccessToken:  fmt.Sprintf("%s-access-token-%d", provider, i),
			RefreshToken: fmt.Sprintf("%s-refresh-token-%d", provider, i),
			IDToken:      fmt.Sprintf("%s-id-token-%d", provider, i),
			Provider:     provider,
			TokenType:    tokenType,
			ExpiresIn:    3600,
		}
	}

	return tokens
}

func generateTestUsers(count int) []TestUser {
	users := make([]TestUser, count)

	for i := range users {
		users[i] = TestUser{
			Username: fmt.Sprintf("user-%d", i),
			Password: fmt.Sprintf("password-%d", i),
		}
	}

	return users
}

func generatePermissionMatrix(userCount, roleCount, resourceCount, permissionTypes int, density float64) *PermissionMatrix {
	matrix := &PermissionMatrix{
		Users:     make([]string, userCount),
		Resources: make([]string, resourceCount),
		Matrix:    make(map[string]map[string]map[string]bool),
	}

	// Generate users and resources
	for i := 0; i < userCount; i++ {
		matrix.Users[i] = fmt.Sprintf("user-%d", i)
	}

	for i := 0; i < resourceCount; i++ {
		matrix.Resources[i] = fmt.Sprintf("resource-%d", i)
	}

	// Generate permission matrix based on density
	permissions := []string{"read", "write", "delete", "create", "update"}

	for _, user := range matrix.Users {
		matrix.Matrix[user] = make(map[string]map[string]bool)

		for _, resource := range matrix.Resources {
			matrix.Matrix[user][resource] = make(map[string]bool)

			for i := 0; i < permissionTypes && i < len(permissions); i++ {
				permission := permissions[i]
				// Randomly grant permission based on density
				matrix.Matrix[user][resource][permission] = (float64(time.Now().UnixNano()%100) / 100.0) < density
			}
		}
	}

	return matrix
}

func generatePermissionChecks(userCount, resourceCount, permissionTypes, count int) []PermissionCheck {
	checks := make([]PermissionCheck, count)
	permissions := []string{"read", "write", "delete", "create", "update"}

	for i := range checks {
		checks[i] = PermissionCheck{
			User:       fmt.Sprintf("user-%d", i%userCount),
			Resource:   fmt.Sprintf("resource-%d", i%resourceCount),
			Permission: permissions[i%min(permissionTypes, len(permissions))],
		}
	}

	return checks
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func setupBenchmarkAuthSystem() *EnhancedAuthSystem {
	config := AuthSystemConfig{
		JWTConfig: JWTConfig{
			SigningMethod: "RS256",
			KeySize:       2048,
		},
		LDAPConfig: LDAPConfig{
			Host:   "localhost:389",
			BaseDN: "dc=example,dc=com",
		},
		OAuth2Providers: []OAuth2Config{
			{Provider: "github", ClientID: "test-client"},
			{Provider: "google", ClientID: "test-client"},
		},
		SessionConfig: SessionConfig{
			TTL:            time.Hour,
			StorageBackend: "memory",
		},
	}

	return NewEnhancedAuthSystem(config)
}

// Helper functions for error checking
func isTokenExpiredError(err error) bool {
	return err != nil && err.Error() == "token is expired"
}

func isSignatureError(err error) bool {
	return err != nil && err.Error() == "signature is invalid"
}

// Enhanced Auth System types and interfaces

type EnhancedAuthSystem struct {
	jwtManager       BenchmarkJWTManager
	rbacEngine       BenchmarkRBACEngine
	ldapClient       BenchmarkLDAPClient
	oauth2Manager    BenchmarkOAuth2Manager
	sessionManager   BenchmarkSessionManager
	tokenCache       BenchmarkTokenCache
	permissionMatrix *PermissionMatrix
	metrics          BenchmarkAuthMetrics
}

type AuthSystemConfig struct {
	JWTConfig       BenchmarkJWTConfig
	LDAPConfig      BenchmarkLDAPConfig
	OAuth2Providers []BenchmarkOAuth2Config
	SessionConfig   BenchmarkSessionConfig
}

type BenchmarkJWTConfig struct {
	SigningMethod string
	KeySize       int
}

type BenchmarkLDAPConfig struct {
	Host               string
	BaseDN             string
	UserSearchBase     string
	GroupSearchBase    string
	ConnectionPoolSize int
	UseConnectionPool  bool
}

type BenchmarkOAuth2Config struct {
	Provider     string
	ClientID     string
	ClientSecret string
	Scopes       []string
	Audience     string
}

type BenchmarkSessionConfig struct {
	TTL             time.Duration
	StorageBackend  string
	CleanupInterval time.Duration
}

type TokenCacheConfig struct {
	MaxSize int
	TTL     time.Duration
}

type BenchmarkRBACConfig struct {
	Roles          []BenchmarkRole
	Permissions    []BenchmarkPermission
	UserGroups     []UserGroup
	HierarchyDepth int
}

type BenchmarkRole struct {
	Name        string
	Permissions []string
}

type BenchmarkPermission struct {
	Name     string
	Resource string
	Action   string
}

type UserGroup struct {
	Name  string
	Users []string
}

type User struct {
	Username   string
	Groups     []string
	Attributes map[string]interface{}
}

type LDAPUser struct {
	Username string
	Password string
	DN       string
	Groups   []string
}

type TestUser struct {
	Username string
	Password string
}

type OAuth2Token struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	Provider     string
	TokenType    string
	ExpiresIn    int
}

type AuthorizationRequest struct {
	Resource string
	Action   string
	Context  map[string]interface{}
}

type PermissionMatrix struct {
	Users     []string
	Resources []string
	Matrix    map[string]map[string]map[string]bool
}

type PermissionCheck struct {
	User       string
	Resource   string
	Permission string
}

// Result types
type JWTValidationResult struct {
	Valid     bool
	Claims    map[string]interface{}
	FromCache bool
}

type AuthorizationResult struct {
	Authorized         bool
	RolesEvaluated     int
	PermissionsChecked int
	CacheHit           bool
}

type LDAPAuthResult struct {
	Authenticated      bool
	BindTime           time.Duration
	SearchTime         time.Duration
	GroupsFound        int
	UsedConnectionPool bool
}

type OAuth2ExchangeResult struct {
	Success         bool
	ValidationTime  time.Duration
	ScopesValidated int
	TokenRefreshed  bool
}

type SessionValidationResult struct {
	Valid     bool
	FromCache bool
	TTL       time.Duration
}

type PermissionEvaluationResult struct {
	Granted       bool
	MatrixLookups int
	UsedCache     bool
}

// Placeholder implementations
func NewEnhancedAuthSystem(config AuthSystemConfig) *EnhancedAuthSystem {
	return &EnhancedAuthSystem{}
}

func (a *EnhancedAuthSystem) Cleanup() {}

func (a *EnhancedAuthSystem) ValidateJWTToken(ctx context.Context, token string) (*JWTValidationResult, error) {
	time.Sleep(100 * time.Microsecond) // Simulate validation time
	return &JWTValidationResult{Valid: true, Claims: map[string]interface{}{"sub": "test"}}, nil
}

func (a *EnhancedAuthSystem) ValidateJWTTokenCached(ctx context.Context, token string) (*JWTValidationResult, error) {
	// Simulate cache hit 70% of the time
	fromCache := (time.Now().UnixNano() % 100) < 70
	if fromCache {
		time.Sleep(10 * time.Microsecond)
	} else {
		time.Sleep(100 * time.Microsecond)
	}
	return &JWTValidationResult{Valid: true, FromCache: fromCache}, nil
}

func (a *EnhancedAuthSystem) ConfigureRBAC(config RBACConfig) {}

func (a *EnhancedAuthSystem) AuthorizeRequest(ctx context.Context, user User, request AuthorizationRequest) (*AuthorizationResult, error) {
	time.Sleep(50 * time.Microsecond)
	return &AuthorizationResult{Authorized: true, RolesEvaluated: 3, PermissionsChecked: 5}, nil
}

func (a *EnhancedAuthSystem) ConfigureLDAP(config LDAPConfig) {}

func (a *EnhancedAuthSystem) AuthenticateLDAP(ctx context.Context, username, password string) (*LDAPAuthResult, error) {
	time.Sleep(20 * time.Millisecond) // Simulate LDAP latency
	return &LDAPAuthResult{
		Authenticated: true,
		BindTime:      5 * time.Millisecond,
		SearchTime:    15 * time.Millisecond,
		GroupsFound:   3,
	}, nil
}

func (a *EnhancedAuthSystem) ConfigureOAuth2(config OAuth2Config) {}

func (a *EnhancedAuthSystem) ExchangeOAuth2Token(ctx context.Context, token OAuth2Token, config OAuth2Config) (*OAuth2ExchangeResult, error) {
	time.Sleep(30 * time.Millisecond) // Simulate OAuth2 exchange
	return &OAuth2ExchangeResult{
		Success:         true,
		ValidationTime:  10 * time.Millisecond,
		ScopesValidated: len(config.Scopes),
	}, nil
}

func (a *EnhancedAuthSystem) ConfigureSessionManagement(config SessionConfig) {}

func (a *EnhancedAuthSystem) CreateSession(ctx context.Context, userID string) (string, error) {
	time.Sleep(1 * time.Millisecond)
	return fmt.Sprintf("session-%s-%d", userID, time.Now().UnixNano()), nil
}

func (a *EnhancedAuthSystem) ValidateSession(ctx context.Context, sessionID string) (*SessionValidationResult, error) {
	time.Sleep(500 * time.Microsecond)
	return &SessionValidationResult{Valid: true, FromCache: true, TTL: time.Hour}, nil
}

func (a *EnhancedAuthSystem) RefreshSession(ctx context.Context, sessionID string) error {
	time.Sleep(2 * time.Millisecond)
	return nil
}

func (a *EnhancedAuthSystem) DeleteSession(ctx context.Context, sessionID string) error {
	time.Sleep(1 * time.Millisecond)
	return nil
}

func (a *EnhancedAuthSystem) ConfigureTokenCache(config TokenCacheConfig) {}

func (a *EnhancedAuthSystem) LoadPermissionMatrix(matrix *PermissionMatrix) {}

func (a *EnhancedAuthSystem) EvaluatePermission(ctx context.Context, check PermissionCheck) (*PermissionEvaluationResult, error) {
	time.Sleep(10 * time.Microsecond)
	return &PermissionEvaluationResult{Granted: true, MatrixLookups: 1, UsedCache: true}, nil
}

// Interface placeholders for benchmarks
type BenchmarkJWTManager interface{}
type BenchmarkRBACEngine interface{}
type BenchmarkLDAPClient interface{}
type BenchmarkOAuth2Manager interface{}
type BenchmarkSessionManager interface{}
type BenchmarkTokenCache interface{}
type BenchmarkAuthMetrics interface{}

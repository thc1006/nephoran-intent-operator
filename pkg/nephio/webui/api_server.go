
package webui



import (

	"context"

	"crypto/tls"

	"fmt"

	"net"

	"net/http"

	"os"

	"os/signal"

	"sync"

	"syscall"

	"time"



	"github.com/gorilla/mux"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.uber.org/zap"

	"golang.org/x/sync/errgroup"



	"k8s.io/client-go/kubernetes"

)



// APIServer represents the Nephio Web UI API server.

type APIServer struct {

	logger       *zap.Logger

	router       *mux.Router

	httpServer   *http.Server

	httpsServer  *http.Server

	kubeClient   kubernetes.Interface

	wsServer     *WebSocketServer

	tlsCertPath  string

	tlsKeyPath   string

	httpPort     int

	httpsPort    int

	stopChan     chan struct{}

	readyChan    chan struct{}

	shutdownWait time.Duration

}



// APIServerConfig provides configuration options for the API server.

type APIServerConfig struct {

	Logger       *zap.Logger

	KubeClient   kubernetes.Interface

	TLSCertPath  string

	TLSKeyPath   string

	HTTPPort     int

	HTTPSPort    int

	ShutdownWait time.Duration

}



// NewAPIServer creates a new Nephio Web UI API server.

func NewAPIServer(config APIServerConfig) *APIServer {

	router := mux.NewRouter()

	wsServer := NewWebSocketServer(config.Logger, config.KubeClient)



	server := &APIServer{

		logger:       config.Logger,

		router:       router,

		kubeClient:   config.KubeClient,

		wsServer:     wsServer,

		tlsCertPath:  config.TLSCertPath,

		tlsKeyPath:   config.TLSKeyPath,

		httpPort:     config.HTTPPort,

		httpsPort:    config.HTTPSPort,

		stopChan:     make(chan struct{}),

		readyChan:    make(chan struct{}),

		shutdownWait: config.ShutdownWait,

	}



	server.setupRoutes()

	return server

}



// setupRoutes configures API routes and middleware.

func (s *APIServer) setupRoutes() {

	// Package handlers.

	packageHandlers := NewPackageHandlers(s.logger, s.kubeClient)

	clusterHandlers := NewClusterHandlers(s.logger, s.kubeClient)

	intentHandlers := NewNetworkIntentHandlers(s.logger, s.kubeClient)

	systemHandlers := NewSystemHandlers(s.logger, s.kubeClient)



	// Base router with common middleware.

	baseRouter := s.router.PathPrefix("/api/v1").Subrouter()

	baseRouter.Use(

		LoggingMiddleware(s.logger),

		RateLimitMiddleware(10, 20),

		SecurityHeadersMiddleware,

	)



	// CORS configuration.

	baseRouter.Use(CORSMiddleware([]string{"*"}))



	// WebSocket endpoint.

	s.router.HandleFunc("/api/v1/events", s.wsServer.HandleWebSocket)



	// Metrics endpoint.

	s.router.Handle("/metrics", promhttp.Handler())



	// Package routes.

	baseRouter.HandleFunc("/packages", packageHandlers.ListPackages).Methods("GET")

	baseRouter.HandleFunc("/packages", packageHandlers.CreatePackage).Methods("POST")

	baseRouter.HandleFunc("/packages/{id}", packageHandlers.GetPackage).Methods("GET")

	baseRouter.HandleFunc("/packages/{id}", packageHandlers.UpdatePackage).Methods("PUT")

	baseRouter.HandleFunc("/packages/{id}", packageHandlers.DeletePackage).Methods("DELETE")



	// Cluster routes.

	baseRouter.HandleFunc("/clusters", clusterHandlers.ListClusters).Methods("GET")



	// Network Intent routes.

	baseRouter.HandleFunc("/intents", intentHandlers.ListIntents).Methods("GET")

	baseRouter.HandleFunc("/intents", intentHandlers.SubmitIntent).Methods("POST")



	// System routes.

	baseRouter.HandleFunc("/health", systemHandlers.GetHealthStatus).Methods("GET")

}



// Start launches the API server.

func (s *APIServer) Start(ctx context.Context) error {

	// Create HTTP server.

	httpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.httpPort))

	if err != nil {

		return fmt.Errorf("failed to listen on HTTP port: %w", err)

	}



	s.httpServer = &http.Server{

		Addr:              httpListener.Addr().String(),

		Handler:           s.router,

		ReadHeaderTimeout: 5 * time.Second,

		IdleTimeout:       120 * time.Second,

	}



	// Create HTTPS server if TLS config is provided.

	var httpsListener net.Listener

	if s.tlsCertPath != "" && s.tlsKeyPath != "" {

		tlsConfig := &tls.Config{

			MinVersion:               tls.VersionTLS12,

			PreferServerCipherSuites: true,

			CipherSuites: []uint16{

				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,

				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,

			},

		}



		httpsListener, err = tls.Listen("tcp", fmt.Sprintf(":%d", s.httpsPort), tlsConfig)

		if err != nil {

			return fmt.Errorf("failed to listen on HTTPS port: %w", err)

		}



		s.httpsServer = &http.Server{

			Addr:              httpsListener.Addr().String(),

			Handler:           s.router,

			ReadHeaderTimeout: 5 * time.Second,

			IdleTimeout:       120 * time.Second,

			TLSConfig:         tlsConfig,

		}

	}



	// Start WebSocket server.

	if err := s.wsServer.Start(ctx); err != nil {

		return fmt.Errorf("failed to start WebSocket server: %w", err)

	}



	// Use errgroup for managing server goroutines.

	eg, serverCtx := errgroup.WithContext(ctx)



	// HTTP server goroutine.

	eg.Go(func() error {

		s.logger.Info("Starting HTTP server", zap.Int("port", s.httpPort))

		if err := s.httpServer.Serve(httpListener); err != http.ErrServerClosed {

			return fmt.Errorf("HTTP server error: %w", err)

		}

		return nil

	})



	// HTTPS server goroutine (if configured).

	if s.httpsServer != nil {

		eg.Go(func() error {

			s.logger.Info("Starting HTTPS server", zap.Int("port", s.httpsPort))

			if err := s.httpsServer.Serve(httpsListener); err != http.ErrServerClosed {

				return fmt.Errorf("HTTPS server error: %w", err)

			}

			return nil

		})

	}



	// Signal handling and graceful shutdown.

	eg.Go(func() error {

		sigChan := make(chan os.Signal, 1)

		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)



		select {

		case sig := <-sigChan:

			s.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

			return s.Shutdown()

		case <-serverCtx.Done():

			return s.Shutdown()

		}

	})



	// Signal server is ready.

	close(s.readyChan)



	return eg.Wait()

}



// Shutdown gracefully stops the API server.

func (s *APIServer) Shutdown() error {

	s.logger.Info("Initiating graceful shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownWait)

	defer cancel()



	var wg sync.WaitGroup

	wg.Add(2)



	go func() {

		defer wg.Done()

		if s.httpServer != nil {

			if err := s.httpServer.Shutdown(ctx); err != nil {

				s.logger.Error("HTTP server shutdown error", zap.Error(err))

			}

		}

	}()



	go func() {

		defer wg.Done()

		if s.httpsServer != nil {

			if err := s.httpsServer.Shutdown(ctx); err != nil {

				s.logger.Error("HTTPS server shutdown error", zap.Error(err))

			}

		}

	}()



	// Stop WebSocket server.

	s.wsServer.Stop()



	wg.Wait()

	close(s.stopChan)



	s.logger.Info("Server shutdown complete")

	return nil

}



// Ready provides a channel that is closed when the server is ready.

func (s *APIServer) Ready() <-chan struct{} {

	return s.readyChan

}



// Stop provides a channel that is closed when the server is stopped.

func (s *APIServer) Stop() <-chan struct{} {

	return s.stopChan

}


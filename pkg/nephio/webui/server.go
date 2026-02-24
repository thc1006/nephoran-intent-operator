package nephiowebui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// Server represents the Web UI server
type Server struct {
	clientset       kubernetes.Interface
	informerFactory informers.SharedInformerFactory
	router          *mux.Router
	port            int
	logger          logr.Logger
}

// NewServer creates a new Web UI server
func NewServer(clientset kubernetes.Interface, port int) *Server {
	informerFactory := informers.NewSharedInformerFactory(clientset, time.Minute*10)

	return &Server{
		clientset:       clientset,
		informerFactory: informerFactory,
		router:          mux.NewRouter(),
		port:            port,
	}
}

// Start starts the Web UI server
func (s *Server) Start(ctx context.Context) error {
	// Start informers
	s.informerFactory.Start(ctx.Done())

	// Wait for informers to sync
	if !cache.WaitForCacheSync(ctx.Done(), s.informerFactory.Core().V1().Pods().Informer().HasSynced) {
		return fmt.Errorf("failed to sync informers")
	}

	// Setup routes
	s.setupRoutes()

	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	if s.logger.Enabled() {
		s.logger.Info("Starting Web UI server", "port", s.port)
	}
	return server.ListenAndServe()
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	s.router.HandleFunc("/api/v1/pods", s.handleGetPods).Methods("GET")
	s.router.HandleFunc("/api/v1/deployments", s.handleGetDeployments).Methods("GET")
	s.router.HandleFunc("/api/v1/services", s.handleGetServices).Methods("GET")
	s.router.HandleFunc("/api/v1/namespaces", s.handleGetNamespaces).Methods("GET")

	// Serve static files
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
}

// handleGetPods handles GET /api/v1/pods
func (s *Server) handleGetPods(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	labelSelector := r.URL.Query().Get("labelSelector")

	podLister := s.informerFactory.Core().V1().Pods().Lister()

	var pods interface{}
	var err error

	if namespace != "" {
		if labelSelector != "" {
			selector, parseErr := labels.Parse(labelSelector)
			if parseErr != nil {
				http.Error(w, fmt.Sprintf("Invalid label selector: %v", parseErr), http.StatusBadRequest)
				return
			}
			pods, err = podLister.Pods(namespace).List(selector)
		} else {
			pods, err = podLister.Pods(namespace).List(labels.Everything())
		}
	} else {
		if labelSelector != "" {
			selector, parseErr := labels.Parse(labelSelector)
			if parseErr != nil {
				http.Error(w, fmt.Sprintf("Invalid label selector: %v", parseErr), http.StatusBadRequest)
				return
			}
			pods, err = podLister.List(selector)
		} else {
			pods, err = podLister.List(labels.Everything())
		}
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list pods: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pods)
}

// handleGetDeployments handles GET /api/v1/deployments
func (s *Server) handleGetDeployments(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")

	deploymentLister := s.informerFactory.Apps().V1().Deployments().Lister()

	var deployments interface{}
	var err error

	if namespace != "" {
		deployments, err = deploymentLister.Deployments(namespace).List(labels.Everything())
	} else {
		deployments, err = deploymentLister.List(labels.Everything())
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list deployments: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deployments)
}

// handleGetServices handles GET /api/v1/services
func (s *Server) handleGetServices(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")

	serviceLister := s.informerFactory.Core().V1().Services().Lister()

	var services interface{}
	var err error

	if namespace != "" {
		services, err = serviceLister.Services(namespace).List(labels.Everything())
	} else {
		services, err = serviceLister.List(labels.Everything())
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list services: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

// handleGetNamespaces handles GET /api/v1/namespaces
func (s *Server) handleGetNamespaces(w http.ResponseWriter, r *http.Request) {
	namespaceLister := s.informerFactory.Core().V1().Namespaces().Lister()

	namespaces, err := namespaceLister.List(labels.Everything())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list namespaces: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(namespaces)
}

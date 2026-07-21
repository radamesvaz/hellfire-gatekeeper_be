package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/joho/godotenv"
	h "github.com/radamesvaz/bakery-app/internal/handlers"
	"github.com/radamesvaz/bakery-app/internal/handlers/auth"
	"github.com/radamesvaz/bakery-app/internal/logger"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	authActionTokensRepo "github.com/radamesvaz/bakery-app/internal/repository/auth_action_tokens"
	bootstrapRepository "github.com/radamesvaz/bakery-app/internal/repository/bootstrap"
	ordersRepository "github.com/radamesvaz/bakery-app/internal/repository/orders"
	productsRepository "github.com/radamesvaz/bakery-app/internal/repository/products"
	tenantRepository "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	tenantSignupRepository "github.com/radamesvaz/bakery-app/internal/repository/tenantsignup"
	"github.com/radamesvaz/bakery-app/internal/repository/user"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	authActionTokensService "github.com/radamesvaz/bakery-app/internal/services/auth_action_tokens"
	bootstrapService "github.com/radamesvaz/bakery-app/internal/services/bootstrap"
	emailService "github.com/radamesvaz/bakery-app/internal/services/email"
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
	invitationService "github.com/radamesvaz/bakery-app/internal/services/invitations"
	orderService "github.com/radamesvaz/bakery-app/internal/services/orders"
	passwordResetService "github.com/radamesvaz/bakery-app/internal/services/passwordreset"
	subscriptionService "github.com/radamesvaz/bakery-app/internal/services/subscriptions"
	tenantSignupService "github.com/radamesvaz/bakery-app/internal/services/tenantsignup"
	tokensService "github.com/radamesvaz/bakery-app/internal/services/tokens"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env when present (local dev). In Docker, env comes from Compose / runtime — no file in the image.
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			fmt.Println("⚠ Could not load .env file:", err)
		}
	}

	// Get log level from environment, default to "info" if not set
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logger.Init(logLevel)

	// Prefer full connection URL if provided (e.g., DATABASE_URL from Supabase/Render)
	databaseURL := os.Getenv("DATABASE_URL")

	// Fallback to discrete variables with PG* compatibility
	dbUser := firstNonEmpty(os.Getenv("POSTGRES_USER"), os.Getenv("PGUSER"))
	dbPassword := firstNonEmpty(os.Getenv("POSTGRES_PASSWORD"), os.Getenv("PGPASSWORD"))
	dbHost := firstNonEmpty(os.Getenv("DB_HOST"), os.Getenv("POSTGRES_HOST"), os.Getenv("PGHOST"))
	dbPort := firstNonEmpty(os.Getenv("DB_PORT"), os.Getenv("POSTGRES_PORT"), os.Getenv("PGPORT"), "5432")
	dbName := firstNonEmpty(os.Getenv("POSTGRES_DB"), os.Getenv("PGDATABASE"))
	secret := os.Getenv("JWT_SECRET")
	expMinutes := os.Getenv("JWT_EXPIRATION_MINUTES")
	port := os.Getenv("PORT")
	// Optional, opt-in toggles to try extra connection candidates
	tryBothPoolers := isTruthy(os.Getenv("DB_TRY_BOTH_POOLERS"))   // tries 5432 and 6543
	tryHostVariants := isTruthy(os.Getenv("DB_TRY_HOST_VARIANTS")) // tries aws-0 and aws-1 variants
	exp, err := strconv.Atoi(expMinutes)
	if err != nil {
		logger.Err(err).Str("env_var", "JWT_EXPIRATION_MINUTES").Msg("could not get the expMinutes from env")
		panic(err)
	}

	// Set default port if not provided
	if port == "" {
		port = "8080"
	}

	// Validate required database configuration
	if databaseURL == "" {
		if dbHost == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			logger.Error().Msg("Missing required database environment variables")
			panic("Database configuration incomplete")
		}
	}

	// Warn if using default database name
	if dbName == "postgres" {
		logger.Warn().Msg("Using default 'postgres' database name. Make sure this is correct for your Supabase setup.")
	}

	// Debug: Show what IPs are being resolved (only when host is known)
	if databaseURL == "" && dbHost != "" {
		logger.Debug().Str("hostname", dbHost).Msg("Resolving hostname")
		ips, err := net.LookupIP(dbHost)
		if err != nil {
			logger.Warn().Err(err).Str("hostname", dbHost).Msg("DNS lookup failed")
		} else {
			ipStrings := make([]string, len(ips))
			for i, ip := range ips {
				ipStrings[i] = ip.String()
			}
			logger.Debug().Strs("ips", ipStrings).Str("hostname", dbHost).Msg("Resolved IPs")
		}
	}

	// Build DSN
	var candidateDSNs []string
	if databaseURL != "" {
		candidateDSNs = append(candidateDSNs, databaseURL)
	}

	// Compose DSNs from discrete vars. Optionally add fallbacks controlled by env flags.
	if dbHost != "" && dbUser != "" && dbPassword != "" && dbName != "" {
		// PGSSLMODE / DB_SSLMODE override heuristic (e.g. PGSSLMODE=disable for Docker Postgres without TLS).
		sslMode := firstNonEmpty(
			strings.TrimSpace(os.Getenv("PGSSLMODE")),
			strings.TrimSpace(os.Getenv("DB_SSLMODE")),
		)
		if sslMode == "" {
			sslMode = "require"
			lowerHost := strings.ToLower(dbHost)
			if lowerHost == "localhost" || lowerHost == "127.0.0.1" || lowerHost == "::1" {
				sslMode = "disable"
			}
		}

		// Build host candidates (no variants unless explicitly enabled)
		hosts := []string{dbHost}
		if tryHostVariants {
			if strings.Contains(dbHost, "aws-1-") {
				hosts = append(hosts, strings.Replace(dbHost, "aws-1-", "aws-0-", 1))
			} else if strings.Contains(dbHost, "aws-0-") {
				hosts = append(hosts, strings.Replace(dbHost, "aws-0-", "aws-1-", 1))
			}
		}

		// Build port candidates (no alternates unless explicitly enabled)
		ports := []string{dbPort}
		if dbPort == "" {
			ports = []string{"5432"}
		}
		if tryBothPoolers {
			otherPort := "5432"
			if len(ports) > 0 && ports[0] == "5432" {
				otherPort = "6543"
			} else {
				otherPort = "5432"
			}
			ports = append(ports, otherPort)
		}

		for _, h := range hosts {
			for _, p := range ports {
				dsn := fmt.Sprintf(
					"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=30 fallback_application_name=hellfire-gatekeeper",
					h, p, dbUser, dbPassword, dbName, sslMode,
				)
				candidateDSNs = append(candidateDSNs, dsn)
			}
		}
	}

	var db *sql.DB
	var lastErr error
	for idx, d := range candidateDSNs {
		// Log a safe summary of the DSN (without password)
		if strings.HasPrefix(d, "postgres://") || strings.HasPrefix(d, "postgresql://") {
			logger.Info().
				Int("candidate", idx+1).
				Int("total_candidates", len(candidateDSNs)).
				Str("method", "DATABASE_URL").
				Msg("Attempting DB connect")
		} else {
			// Parse host/port for logging
			var host, portStr, dbn string
			host = dbHost
			portStr = dbPort
			dbn = dbName
			logger.Info().
				Int("candidate", idx+1).
				Int("total_candidates", len(candidateDSNs)).
				Str("host", host).
				Str("port", portStr).
				Str("user", dbUser).
				Str("dbname", dbn).
				Msg("Attempting DB connect")
		}

		db, err = sql.Open("postgres", d)
		if err != nil {
			logger.Err(err).
				Int("candidate", idx+1).
				Msg("sql.Open failed")
			lastErr = err
			continue
		}

		// Try ping with backoff for this candidate
		rand.Seed(time.Now().UnixNano())
		maxAttempts := 5
		baseDelay := 1 * time.Second
		succeeded := false
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			if err := db.Ping(); err != nil {
				logger.Warn().Err(err).
					Int("candidate", idx+1).
					Int("attempt", attempt).
					Int("max_attempts", maxAttempts).
					Msg("Could not ping database")
				lastErr = err
				if attempt == maxAttempts {
					break
				}
				exp := baseDelay * time.Duration(1<<uint(attempt-1))
				if exp > 5*time.Second {
					exp = 5 * time.Second
				}
				jitter := time.Duration(rand.Int63n(int64(exp / 5)))
				time.Sleep(exp + jitter)
				continue
			}
			logger.Info().
				Int("candidate", idx+1).
				Int("attempt", attempt).
				Msg("Database connected successfully")
			succeeded = true
			break
		}

		if succeeded {
			break
		}

		// Close and move to next candidate
		_ = db.Close()
		db = nil
	}

	if db == nil {
		logger.Logger.Fatal().Err(lastErr).
			Int("total_candidates", len(candidateDSNs)).
			Msg("Could not connect to the DB after trying all candidates")
		panic(lastErr)
	}

	// Configure connection pool for production stability (overridable via env)
	maxOpen := parseIntWithDefault(os.Getenv("DB_MAX_OPEN_CONNS"), 10)
	maxIdle := parseIntWithDefault(os.Getenv("DB_MAX_IDLE_CONNS"), 5)
	maxLifetimeMin := parseIntWithDefault(os.Getenv("DB_CONN_MAX_LIFETIME_MIN"), 5)
	maxIdleTimeMin := parseIntWithDefault(os.Getenv("DB_CONN_MAX_IDLE_TIME_MIN"), 1)

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(maxLifetimeMin) * time.Minute)
	db.SetConnMaxIdleTime(time.Duration(maxIdleTimeMin) * time.Minute)

	// Test connection with exponential backoff + jitter (tolerates cold start/pooler warmup)
	rand.Seed(time.Now().UnixNano())
	maxAttempts := 10
	baseDelay := 1 * time.Second
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := db.Ping(); err != nil {
			logger.Warn().Err(err).
				Int("attempt", attempt).
				Int("max_attempts", maxAttempts).
				Msg("Could not ping database")
			if attempt == maxAttempts {
				panic(err)
			}
			// Exponential backoff capped at 10s, add +/-20%% jitter
			exp := baseDelay * time.Duration(1<<uint(attempt-1))
			if exp > 10*time.Second {
				exp = 10 * time.Second
			}
			jitter := time.Duration(rand.Int63n(int64(exp / 5))) // up to 20%
			sleepFor := exp + jitter
			time.Sleep(sleepFor)
			continue
		}
		logger.Info().Msg("Database connected successfully")
		break
	}
	defer db.Close()

	// Product setup
	productRepo := &productsRepository.ProductRepository{DB: db}

	// Image service setup
	uploadDir := "uploads"
	imageService := imagesService.New(uploadDir)

	// Product handler (only for product data)
	productHandler := &h.ProductHandler{
		Repo: productRepo,
	}

	// Image handler (only for image management)
	imageHandler := &h.ImageHandler{
		Repo:         productRepo,
		ImageService: imageService,
	}

	// Tenant repo (tenant name on register; slug resolution for public routes)
	tenantRepo := &tenantRepository.Repository{DB: db}

	// Auth setup: construct token managers once and share the same OneTimeTokenManager instance
	// across AuthService and future flows (tenant OTC, invitations, etc.).
	sessionTokenManager := tokensService.NewJWTSessionTokenManager(secret, exp)
	oneTimeTokenManager := tokensService.NewSHA256OneTimeTokenManager(32)
	authSvc := authService.NewWithManagers(sessionTokenManager, oneTimeTokenManager)

	userRepo := user.UserRepository{DB: db}
	enableTenantRegister := parseBoolWithDefault(os.Getenv("ENABLE_TENANT_REGISTER"), true)
	// Login + /register use authSvc for session JWT; same instance backs tenant signup OTC via TenantSignupService.
	authHandler := &auth.LoginHandler{
		UserRepo:              userRepo,
		TenantRepo:            tenantRepo,
		AuthService:           authSvc,
		TenantRegisterEnabled: &enableTenantRegister,
	}
	bootstrapRepo := &bootstrapRepository.Repository{DB: db}
	bootstrapSvc := &bootstrapService.BootstrapService{
		Repo:        bootstrapRepo,
		AuthService: authSvc,
	}
	bootstrapHandler := &auth.BootstrapHandler{
		Service: bootstrapSvc,
	}
	tenantSignupRepo := &tenantSignupRepository.Repository{DB: db}
	tenantSignupSvc := &tenantSignupService.TenantSignupService{
		Repo:        tenantSignupRepo,
		AuthService: authSvc,
		EmailSender: resolveEmailSender(),
		AppBaseURL:  strings.TrimSpace(os.Getenv("APP_BASE_URL")),
	}
	tenantSignupHandler := &auth.TenantSignupHandler{
		Service: tenantSignupSvc,
	}
	subscriptionGraceDays := parseIntWithDefault(os.Getenv("SUBSCRIPTION_GRACE_DAYS"), subscriptionService.DefaultGraceDays)
	subscriptionSvc := subscriptionService.NewService(tenantRepo, subscriptionGraceDays)
	subscriptionHandler := &auth.SubscriptionHandler{
		Service: subscriptionSvc,
	}
	actionTokensRepoInst := &authActionTokensRepo.SQLRepository{DB: db}
	authActionTokensSvc := &authActionTokensService.ActionTokenService{
		DB:          db,
		Repo:        actionTokensRepoInst,
		AuthService: authSvc,
	}
	passwordResetSvc := &passwordResetService.PasswordResetService{
		Users:        &userRepo,
		AuthService:  authSvc,
		TokenService: authActionTokensSvc,
		EmailSender:  resolveEmailSender(),
		AppBaseURL:   strings.TrimSpace(os.Getenv("APP_BASE_URL")),
	}
	passwordResetHandler := &auth.PasswordResetHandler{
		Service: passwordResetSvc,
	}
	invitationSvc := &invitationService.InvitationService{
		Users:        &userRepo,
		AuthService:  authSvc,
		TokenService: authActionTokensSvc,
		EmailSender:  resolveEmailSender(),
		AppBaseURL:   strings.TrimSpace(os.Getenv("APP_BASE_URL")),
	}
	invitationHandler := &auth.InvitationHandler{
		Service: invitationSvc,
	}

	tenantHandler := &h.TenantHandler{
		Repo:         tenantRepo,
		ImageService: imageService,
	}

	// Order setup
	orderRepo := &ordersRepository.OrderRepository{DB: db}
	orderHandler := &h.OrderHandler{
		Repo:        orderRepo,
		UserRepo:    &userRepo,
		ProductRepo: productRepo,
		TenantRepo:  tenantRepo,
	}

	// Ghost order worker: cancel expired pending orders on an interval
	ghostOrderIntervalMin := parseIntWithDefault(os.Getenv("GHOST_ORDER_CRON_INTERVAL_MINUTES"), 5)
	ghostCanceller := orderService.NewExpiredOrderCanceller(orderRepo, productRepo, tenantRepo)
	subscriptionIntervalHours := parseIntWithDefault(os.Getenv("SUBSCRIPTION_CRON_INTERVAL_HOURS"), 24)
	workerCtx, workerCancel := context.WithCancel(context.Background())
	var workerWg sync.WaitGroup
	workerWg.Add(1)
	go func() {
		defer workerWg.Done()
		orderService.RunGhostOrderWorker(workerCtx, ghostCanceller, ghostOrderIntervalMin)
	}()
	workerWg.Add(1)
	go func() {
		defer workerWg.Done()
		subscriptionService.RunWorker(workerCtx, subscriptionSvc, subscriptionIntervalHours)
	}()

	r := mux.NewRouter()
	rateLimiter := middleware.NewInMemoryRateLimiter()
	rateLimitWindow := time.Duration(parseIntWithDefault(os.Getenv("RATE_LIMIT_WINDOW_SECONDS"), 60)) * time.Second
	forgotRateLimit := rateLimiter.Middleware(middleware.RateLimitOptions{
		Name:        "password_forgot",
		MaxRequests: parseIntWithDefault(os.Getenv("RATE_LIMIT_FORGOT_MAX"), 5),
		Window:      rateLimitWindow,
		ScopeTenant: true,
	})
	resetRateLimit := rateLimiter.Middleware(middleware.RateLimitOptions{
		Name:        "password_reset",
		MaxRequests: parseIntWithDefault(os.Getenv("RATE_LIMIT_RESET_MAX"), 10),
		Window:      rateLimitWindow,
		ScopeTenant: true,
	})
	inviteAcceptRateLimit := rateLimiter.Middleware(middleware.RateLimitOptions{
		Name:        "invite_accept",
		MaxRequests: parseIntWithDefault(os.Getenv("RATE_LIMIT_INVITE_ACCEPT_MAX"), 10),
		Window:      rateLimitWindow,
		ScopeTenant: true,
	})
	inviteCreateRateLimit := rateLimiter.Middleware(middleware.RateLimitOptions{
		Name:        "invite_create",
		MaxRequests: parseIntWithDefault(os.Getenv("RATE_LIMIT_INVITE_CREATE_MAX"), 10),
		Window:      rateLimitWindow,
		ScopeTenant: true,
		ScopeUser:   true,
	})
	inviteResendRateLimit := rateLimiter.Middleware(middleware.RateLimitOptions{
		Name:        "invite_resend",
		MaxRequests: parseIntWithDefault(os.Getenv("RATE_LIMIT_INVITE_RESEND_MAX"), 5),
		Window:      rateLimitWindow,
		ScopeTenant: true,
		ScopeUser:   true,
	})

	// CORS configuration (allowlist + credentials)
	allowedOrigins := handlers.AllowedOrigins([]string{
		"http://localhost:5173",
		"http://localhost:3000",
		"http://localhost:5000",
		"https://confettideliadmin.netlify.app",
		"https://confettideli.netlify.app",
	})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	allowedHeaders := handlers.AllowedHeaders([]string{
		"Authorization",
		"Content-Type",
		"X-Requested-With",
		"Accept",
		"Origin",
		"Access-Control-Request-Method",
		"Access-Control-Request-Headers",
		"X-Tenant-Slug",
	})
	allowCredentials := handlers.AllowCredentials()

	// Serve static files (images)
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Database connection failed: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}).Methods("GET")

	r.HandleFunc("/products", productHandler.GetAllProducts).Methods("GET")
	r.HandleFunc("/products/{id}", productHandler.GetProductByID).Methods("GET")
	r.HandleFunc("/setup/bootstrap/tenant", bootstrapHandler.BootstrapTenant).Methods("POST")
	r.HandleFunc("/public/tenant-register", tenantSignupHandler.RegisterTenantWithCode).Methods("POST")
	// Auth endpoints (legacy single-tenant)
	r.HandleFunc("/login", authHandler.Login).Methods("POST")
	r.HandleFunc("/register", authHandler.Register).Methods("POST")

	// Auth endpoints (multi-tenant, path-based)
	tAuth := r.PathPrefix("/t/{tenant_slug}/auth").Subrouter()
	tAuth.Use(middleware.TenantFromPathOrHeader(tenantRepo))
	tAuth.HandleFunc("/login", authHandler.Login).Methods("POST")
	tAuth.HandleFunc("/register", authHandler.Register).Methods("POST")
	tAuth.Handle("/password/forgot", forgotRateLimit(http.HandlerFunc(passwordResetHandler.ForgotPassword))).Methods("POST")
	tAuth.Handle("/password/reset", resetRateLimit(http.HandlerFunc(passwordResetHandler.ResetPassword))).Methods("POST")
	tAuth.Handle("/invitations/accept", inviteAcceptRateLimit(http.HandlerFunc(invitationHandler.AcceptInvitation))).Methods("POST")

	// Invitation admin under path tenant (slug from URL). Inherits TenantFromPathOrHeader from tAuth; JWT must match path tenant.
	tAuthInv := tAuth.PathPrefix("/invitations").Subrouter()
	tAuthInv.Use(middleware.AuthMiddleware(authSvc))
	tAuthInv.Use(middleware.RequireJWTTenantMatchesContext())
	tAuthInv.Handle("", inviteCreateRateLimit(http.HandlerFunc(invitationHandler.CreateInvitation))).Methods("POST")
	tAuthInv.Handle("/{id}/revoke", http.HandlerFunc(invitationHandler.RevokeInvitation)).Methods("POST")
	tAuthInv.Handle("/{id}/resend", inviteResendRateLimit(http.HandlerFunc(invitationHandler.ResendInvitation))).Methods("POST")

	// Authenticated API: JWT + tenant context (path slug if present, else slug from DB via tenant_id claim).
	auth := r.PathPrefix("/auth").Subrouter()
	auth.Use(middleware.AuthMiddleware(authSvc))
	auth.Use(middleware.TenantMiddleware(tenantRepo))
	auth.Use(middleware.RequireOperableSubscription(tenantRepo))
	auth.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Token válido, acceso permitido")
	}).Methods("GET")

	// TODO: cuando exista login de clientes, agrupar en subrouter con middleware de rol admin (tras AuthMiddleware):
	//   POST   /auth/products
	//   PUT    /auth/products/{id}
	//   PATCH  /auth/products/{id}
	//   PATCH  /auth/products/{id}/thumbnail
	//   POST   /auth/products/{id}/thumbnail
	//   POST   /auth/products/{id}/images
	//   PUT    /auth/products/{id}/images
	//   DELETE /auth/products/{id}/images
	//   PATCH  /auth/branding/logo
	//   PATCH  /auth/branding/colors
	//   PATCH  /auth/branding/name
	//   POST   /auth/invitations
	//   POST   /auth/invitations/{id}/revoke
	//   POST   /auth/invitations/{id}/resend
	//   POST   /t/{tenant_slug}/auth/invitations
	//   POST   /t/{tenant_slug}/auth/invitations/{id}/revoke
	//   POST   /t/{tenant_slug}/auth/invitations/{id}/resend
	// Product endpoints (data only)
	auth.HandleFunc("/products", productHandler.CreateProduct).Methods("POST")
	auth.HandleFunc("/products/{id}", productHandler.UpdateProduct).Methods("PUT")
	auth.HandleFunc("/products/{id}", productHandler.UpdateProductStatus).Methods("PATCH")
	auth.HandleFunc("/products/{id}/thumbnail", productHandler.UpdateProductThumbnail).Methods("PATCH")
	auth.HandleFunc("/products/{id}/thumbnail", imageHandler.UploadProductThumbnail).Methods("POST")

	// Image endpoints (image management)
	auth.HandleFunc("/products/{id}/images", imageHandler.AddProductImages).Methods("POST")
	auth.HandleFunc("/products/{id}/images", imageHandler.ReplaceProductImages).Methods("PUT")
	auth.HandleFunc("/products/{id}/images", imageHandler.DeleteProductImage).Methods("DELETE")

	// Order endpoints (authenticated: list, get, update)
	auth.HandleFunc("/orders", orderHandler.GetAllOrders).Methods("GET")
	auth.HandleFunc("/orders/{id}", orderHandler.GetOrderByID).Methods("GET")
	auth.HandleFunc("/orders/{id}", orderHandler.UpdateOrder).Methods("PATCH")

	// Tenant branding: reads are public (see tPublic); mutations require auth
	auth.HandleFunc("/branding/logo", tenantHandler.UploadTenantLogo).Methods("PATCH")
	auth.HandleFunc("/branding/colors", tenantHandler.UpdateBrandingColors).Methods("PATCH")
	auth.HandleFunc("/branding/name", tenantHandler.UpdateTenantDisplayName).Methods("PATCH")
	auth.HandleFunc("/subscription", subscriptionHandler.GetSubscription).Methods("GET")

	authInv := auth.PathPrefix("/invitations").Subrouter()
	authInv.Handle("", inviteCreateRateLimit(http.HandlerFunc(invitationHandler.CreateInvitation))).Methods("POST")
	authInv.Handle("/{id}/revoke", http.HandlerFunc(invitationHandler.RevokeInvitation)).Methods("POST")
	authInv.Handle("/{id}/resend", inviteResendRateLimit(http.HandlerFunc(invitationHandler.ResendInvitation))).Methods("POST")

	// Internal auth API: superadmin operations. Intentionally not gated by RequireOperableSubscription
	// so canceled tenants can be reactivated.
	authInternal := r.PathPrefix("/auth/internal").Subrouter()
	authInternal.Use(middleware.AuthMiddleware(authSvc))
	authInternal.Use(middleware.TenantMiddleware(tenantRepo))
	authInternal.HandleFunc("/tenants/{tenant_id}/subscription", subscriptionHandler.UpdateTenantSubscriptionInternal).Methods("PATCH")
	authInternal.HandleFunc("/tenant-signup-codes", tenantSignupHandler.CreateSignupCode).Methods("POST")

	// Public catalog + orders: tenant from path or X-Tenant-Slug header
	tPublic := r.PathPrefix("/t/{tenant_slug}").Subrouter()
	tPublic.Use(middleware.TenantFromPathOrHeader(tenantRepo))
	tPublic.HandleFunc("/products", productHandler.GetAllProducts).Methods("GET")
	tPublic.HandleFunc("/products/{id}", productHandler.GetProductByID).Methods("GET")
	tPublic.HandleFunc("/branding", tenantHandler.GetBranding).Methods("GET")
	tPublic.HandleFunc("/orders", orderHandler.CreateOrder).Methods("POST")

	// Legacy: POST /orders (no tenant in path; handler falls back to tenant 1)
	r.HandleFunc("/orders", orderHandler.CreateOrder).Methods("POST")

	// Wrap router with CORS
	corsWrapped := handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders, allowCredentials)(r)

	// Start a goroutine to periodically check database health
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := db.Ping(); err != nil {
				logger.Warn().Err(err).Msg("Database health check failed")
			}
		}
	}()

	srv := &http.Server{Addr: ":" + port, Handler: corsWrapped}
	go func() {
		logger.Info().
			Str("port", port).
			Str("address", fmt.Sprintf("http://localhost:%s", port)).
			Msg("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal().Err(err).Msg("Server failed")
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info().Msg("Shutting down: stopping background workers")
	workerCancel()
	workerWg.Wait()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Warn().Err(err).Msg("Server shutdown had an error")
	}
	logger.Info().Msg("Server stopped")
}

// firstNonEmpty returns the first non-empty string from the provided list.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// isTruthy interprets common truthy strings for feature flags.
func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "t", "true", "y", "yes", "on":
		return true
	default:
		return false
	}
}

// parseIntWithDefault converts a string to int, returning defaultValue on error/empty
func parseIntWithDefault(value string, defaultValue int) int {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return v
}

// parseBoolWithDefault converts a string to bool, returning defaultValue on error/empty.
func parseBoolWithDefault(value string, defaultValue bool) bool {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	v, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return v
}

func resolveEmailSender() emailService.Sender {
	apiKey := stripEnvQuotes(strings.TrimSpace(os.Getenv("BREVO_API_KEY")))
	fromEmail := stripEnvQuotes(strings.TrimSpace(os.Getenv("BREVO_FROM_EMAIL")))
	fromName := stripEnvQuotes(strings.TrimSpace(os.Getenv("BREVO_FROM_NAME")))

	if apiKey == "" || fromEmail == "" {
		logger.Warn().Msg("Brevo sender not configured, using noop email sender")
		return emailService.NoopSender{}
	}
	if fromName == "" {
		fromName = "Hellfire Gatekeeper"
	}
	logger.Info().
		Int("brevo_api_key_len", len(apiKey)).
		Bool("brevo_api_key_looks_valid", strings.HasPrefix(apiKey, "xkeysib-")).
		Str("brevo_from_email", fromEmail).
		Msg("Brevo setup successfully")
	return emailService.NewBrevoSender(apiKey, fromEmail, fromName)
}

func stripEnvQuotes(v string) string {
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}

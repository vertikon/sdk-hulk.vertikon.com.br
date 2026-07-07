package hulk

import (
	"context"
	"errors"
	"fmt"
	std_http "net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/ai"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/config"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/events"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/http"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/secrets"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/security"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/secrets/providers/file"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/secrets/providers/vault"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/state"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/telemetry"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
)

// moduleStatus guarda o estado de ciclo de vida de um módulo (observabilidade).
type moduleStatus struct {
	Phase     string    `json:"phase"` // registered|initialized|running|restarting|error|stopped
	Err       string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
}

// App é o container principal do Monolito Modular.
type App struct {
	modules   []Module
	logger    *zap.Logger
	startedAt time.Time
	hulkCtx   Context
	statusMu  sync.RWMutex
	statuses  map[string]*moduleStatus
}

// NewApp cria uma nova instância da aplicação.
func NewApp() *App {
	logger, _ := zap.NewDevelopment()
	return &App{
		modules:  []Module{},
		logger:   logger,
		statuses: map[string]*moduleStatus{},
	}
}

// setStatus atualiza (thread-safe) o estado de um módulo.
func (a *App) setStatus(id, phase, errStr string, startedAt time.Time) {
	a.statusMu.Lock()
	defer a.statusMu.Unlock()
	st, ok := a.statuses[id]
	if !ok {
		st = &moduleStatus{}
		a.statuses[id] = st
	}
	st.Phase = phase
	st.Err = errStr
	if !startedAt.IsZero() {
		st.StartedAt = startedAt
	}
}

// getStatus lê (thread-safe) o estado de um módulo.
func (a *App) getStatus(id string) moduleStatus {
	a.statusMu.RLock()
	defer a.statusMu.RUnlock()
	if st, ok := a.statuses[id]; ok {
		return *st
	}
	return moduleStatus{Phase: "unknown"}
}

// safeLifecycle executa um passo de ciclo de vida (Init/Start) capturando panics,
// para que uma falha não derrube o processo (o chamador decide Fatal vs. continuar).
func safeLifecycle(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn()
}

// findModule localiza um módulo registrado pelo seu ID.
func (a *App) findModule(id string) Module {
	for _, m := range a.modules {
		if m.Config().ID == id {
			return m
		}
	}
	return nil
}

// Register adiciona um módulo à aplicação.
func (a *App) Register(m Module) {
	a.modules = append(a.modules, m)
}

func isProduction(environment string) bool {
	return strings.EqualFold(strings.TrimSpace(environment), "production")
}

func buildLogger(environment, logLevel string) (*zap.Logger, error) {
	env := strings.TrimSpace(environment)
	isProd := isProduction(env)

	var cfg zap.Config
	if isProd {
		cfg = zap.NewProductionConfig()
		cfg.DisableStacktrace = true
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	if logLevel != "" {
		var lvl zapcore.Level
		if err := lvl.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(logLevel)))); err == nil {
			cfg.Level.SetLevel(lvl)
		}
	}

	return cfg.Build()
}

func collectorURLFromEnv() string {
	raw := os.Getenv("OTEL_COLLECTOR_URL")
	if raw == "" {
		raw = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimPrefix(raw, "https://")
	if raw == "" {
		raw = "localhost:4317"
	}
	return raw
}

func sampleRatioFromEnv(isProd bool) float64 {
	ratio := 1.0
	if isProd {
		ratio = 0.1
	}

	if v := strings.TrimSpace(os.Getenv("OTEL_SAMPLE_RATIO")); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	if v := strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER_ARG")); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	return ratio
}

// Run inicia a aplicação e todos os módulos registrados.
func (a *App) Run() {
	a.logger.Info("🚀 Iniciando Vertikon Endurance Monolith...")

	// 1. Carregar Configuração
	cfg, err := config.Load()
	if err != nil {
		a.logger.Fatal("Falha ao carregar configuração", zap.Error(err))
	}

	// 1.1 Logger por ambiente
	if logger, err := buildLogger(cfg.App.Environment, cfg.App.LogLevel); err == nil {
		a.logger = logger
		defer func() { _ = a.logger.Sync() }()
	} else {
		a.logger.Warn("⚠️ Falha ao inicializar logger configurado; usando default", zap.Error(err))
	}
	a.logger.Info("Configuração carregada", zap.String("env", cfg.App.Environment))

	isProd := isProduction(cfg.App.Environment)

	// 1.2 Gate de segredo JWT (fail-fast em produção).
	// Em prod, abortamos AQUI — cedo e com 1 mensagem clara — se nenhum segredo JWT estiver
	// definido, em vez de deixar módulos individuais falharem o Start() no meio do boot (Fatal
	// espalhado). Em dev seguimos com WARN; os módulos usam placeholder de desenvolvimento.
	if jwt := strings.TrimSpace(os.Getenv("HULK_JWT_SECRET")); jwt == "" {
		if jwt = strings.TrimSpace(os.Getenv("JWT_SECRET")); jwt == "" {
			if isProd {
				a.logger.Fatal("Segredo JWT ausente em produção: defina HULK_JWT_SECRET (ou JWT_SECRET) no ambiente antes de iniciar o eduue-api")
			}
			a.logger.Warn("⚠️ Segredo JWT ausente (HULK_JWT_SECRET/JWT_SECRET): usando placeholder de desenvolvimento. NÃO use em produção.")
		}
	}

	// 2. Inicializar Dependências
	ctx := context.Background()

	// --- [BLOCO-P] Inicializar Telemetria (OpenTelemetry) ---
	otelShutdown, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName:    "vertikon-monolith",
		ServiceVersion: "1.0.0",
		Environment:    cfg.App.Environment,
		CollectorURL:   collectorURLFromEnv(),
		SampleRatio:    sampleRatioFromEnv(isProd),
	})
	if err != nil {
		a.logger.Warn("⚠️ Falha ao inicializar OpenTelemetry. Continuando sem observabilidade.", zap.Error(err))
	} else {
		a.logger.Info("✅ OpenTelemetry inicializado")
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := otelShutdown(shutdownCtx); err != nil {
				a.logger.Error("Erro ao desligar OTel", zap.Error(err))
			}
		}()
	}

	// Store (Postgres + Redis)
	if isProd {
		if cfg.Database.Host == "" || cfg.Database.User == "" || cfg.Database.DBName == "" {
			a.logger.Fatal("Config de banco incompleta em produção",
				zap.String("host", cfg.Database.Host),
				zap.String("user", cfg.Database.User),
				zap.String("dbname", cfg.Database.DBName),
			)
		}
	}

	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("VERTIKON_DATABASE_URL"))
	}
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName, cfg.Database.SSLMode)
	}

	var store state.Store
	redisAddr := ""
	if cfg.Redis.URL != "" {
		redisAddr = strings.TrimPrefix(cfg.Redis.URL, "redis://")
		if redisAddr == "" {
			redisAddr = cfg.Redis.URL
		}
	}
	if redisAddr == "" && !isProd {
		redisAddr = "localhost:6379"
	}

	pgStore, err := state.NewPostgresStoreWithRedis(ctx, dsn, redisAddr)
	if err != nil {
		if isProd {
			a.logger.Fatal("Falha ao conectar ao PostgreSQL em produção", zap.Error(err))
		}
		a.logger.Warn("⚠️ Não foi possível conectar ao Banco de Dados. Usando NoOpStore com erro.", zap.Error(err))
		store = &NoOpStore{err: err}
	} else {
		a.logger.Info("✅ Conectado ao PostgreSQL")
		if pgStore.RedisClient() != nil {
			a.logger.Info("✅ Redis habilitado para cache")
		} else {
			a.logger.Warn("⚠️ Redis não disponível. Continuando sem cache.")
		}
		store = pgStore
		defer pgStore.Close()
	}

	// Bus (NATS)
	var bus events.Bus
	if isProd && cfg.NATS.URL == "" {
		a.logger.Fatal("NATS_URL é obrigatório em produção")
	}
	natsBus, err := events.NewNatsBus(cfg.NATS.URL, cfg.App.Environment)
	if err != nil {
		if isProd {
			a.logger.Fatal("Falha ao conectar ao NATS em produção", zap.Error(err))
		}
		a.logger.Warn("⚠️ Não foi possível conectar ao NATS. Usando NoOpBus com erro.", zap.Error(err))
		bus = &NoOpBus{err: err}
	} else {
		a.logger.Info("✅ Conectado ao NATS JetStream")
		bus = natsBus
		defer natsBus.Close()
	}

	// AI (Placeholder por enquanto)
	aiClient := &NoOpAI{}

	// HTTP Server (Echo)
	httpServer := http.NewEchoServer()

	// Default Root Handler (Health Check / Welcome)
	httpServer.GET("/", func(c http.Context) error {
		return c.JSON(std_http.StatusOK, map[string]string{
			"status":  "online",
			"app":     "Vertikon Endurance Monolith",
			"version": "1.0.0",
		})
	})

	a.startedAt = time.Now()

	// Registro de módulos (config + saúde real) — superfície pública de descoberta.
	// Consumida pelo painel do gestor (Ecossistema) para listar módulos e status.
	httpServer.GET("/system/modules", func(c http.Context) error {
		type modInfo struct {
			ID        string    `json:"id"`
			Name      string    `json:"name"`
			Version   string    `json:"version"`
			Status    string    `json:"status"`
			StartedAt time.Time `json:"started_at,omitempty"`
		}
		mods := make([]modInfo, 0, len(a.modules))
		for _, m := range a.modules {
			cfg := m.Config()
			st := a.getStatus(cfg.ID)
			mods = append(mods, modInfo{ID: cfg.ID, Name: cfg.Name, Version: cfg.Version, Status: st.Phase, StartedAt: st.StartedAt})
		}
		return c.JSON(std_http.StatusOK, map[string]interface{}{"count": len(mods), "modules": mods})
	})

	// Saúde agregada (serviços ativos + uptime).
	httpServer.GET("/system/health", func(c http.Context) error {
		running := 0
		for _, m := range a.modules {
			if a.getStatus(m.Config().ID).Phase == "running" {
				running++
			}
		}
		uptime := 0
		if !a.startedAt.IsZero() {
			uptime = int(time.Since(a.startedAt).Seconds())
		}
		return c.JSON(std_http.StatusOK, map[string]interface{}{
			"status":          "ok",
			"started_at":      a.startedAt,
			"uptime_seconds":  uptime,
			"modules_total":   len(a.modules),
			"modules_running": running,
		})
	})

	// Reiniciar um módulo (Stop+Start). PROTEGIDO por HULK_ADMIN_TOKEN e blindado
	// por recover() — nunca derruba o processo. ATENÇÃO: módulos que registram
	// rotas HTTP no Start podem falhar ao re-registrar (limitação do Modulith);
	// o erro é retornado, não propagado.
	httpServer.POST("/system/modules/:id/restart", func(c http.Context) error {
		adminToken := strings.TrimSpace(os.Getenv("HULK_ADMIN_TOKEN"))
		if adminToken == "" {
			return c.JSON(std_http.StatusForbidden, map[string]string{"error": "restart desabilitado: defina HULK_ADMIN_TOKEN"})
		}
		if c.Request().Header.Get("X-Admin-Token") != adminToken {
			return c.JSON(std_http.StatusUnauthorized, map[string]string{"error": "token admin inválido"})
		}
		id := c.Param("id")
		m := a.findModule(id)
		if m == nil {
			return c.JSON(std_http.StatusNotFound, map[string]string{"error": "módulo não encontrado: " + id})
		}
		var restartErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					restartErr = fmt.Errorf("panic no restart: %v", r)
				}
			}()
			a.setStatus(id, "restarting", "", time.Time{})
			stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := m.Stop(stopCtx); err != nil {
				restartErr = fmt.Errorf("stop: %w", err)
				return
			}
			if err := m.Start(a.hulkCtx); err != nil {
				restartErr = fmt.Errorf("start: %w", err)
			}
		}()
		if restartErr != nil {
			a.setStatus(id, "error", restartErr.Error(), time.Time{})
			a.logger.Warn("Falha ao reiniciar módulo", zap.String("module", id), zap.Error(restartErr))
			return c.JSON(std_http.StatusInternalServerError, map[string]interface{}{"id": id, "ok": false, "error": restartErr.Error()})
		}
		a.setStatus(id, "running", "", time.Now())
		a.logger.Info("Módulo reiniciado", zap.String("module", id))
		return c.JSON(std_http.StatusOK, map[string]interface{}{"id": id, "ok": true})
	})

	// Gerador de token JWT (integração/teste). Assina com o segredo do eduue —
	// PROTEGIDO por HULK_ADMIN_TOKEN. O token vale para o middleware híbrido.
	httpServer.POST("/system/dev-token", func(c http.Context) error {
		adminToken := strings.TrimSpace(os.Getenv("HULK_ADMIN_TOKEN"))
		if adminToken == "" {
			return c.JSON(std_http.StatusForbidden, map[string]string{"error": "gerador de token desabilitado: defina HULK_ADMIN_TOKEN"})
		}
		if c.Request().Header.Get("X-Admin-Token") != adminToken {
			return c.JSON(std_http.StatusUnauthorized, map[string]string{"error": "token admin inválido"})
		}
		secret := strings.TrimSpace(os.Getenv("HULK_JWT_SECRET"))
		if secret == "" {
			secret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
		}
		if secret == "" {
			return c.JSON(std_http.StatusInternalServerError, map[string]string{"error": "JWT secret não configurado no backend"})
		}
		var dto struct {
			UserID   string `json:"user_id"`
			Email    string `json:"email"`
			Role     string `json:"role"`
			TTLHours int    `json:"ttl_hours"`
		}
		_ = c.Bind(&dto)
		if dto.UserID == "" {
			dto.UserID = "dev-user"
		}
		if dto.Email == "" {
			dto.Email = "dev@eduue.com.br"
		}
		if dto.Role == "" {
			dto.Role = "admin"
		}
		if dto.TTLHours <= 0 {
			dto.TTLHours = 24
		}
		now := time.Now()
		exp := now.Add(time.Duration(dto.TTLHours) * time.Hour)
		claims := security.HulkClaims{
			UserID: dto.UserID,
			Email:  dto.Email,
			Role:   dto.Role,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(exp),
				IssuedAt:  jwt.NewNumericDate(now),
				NotBefore: jwt.NewNumericDate(now),
			},
		}
		signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
		if err != nil {
			return c.JSON(std_http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(std_http.StatusOK, map[string]interface{}{
			"token":      signed,
			"token_type": "Bearer",
			"user_id":    dto.UserID,
			"email":      dto.Email,
			"role":       dto.Role,
			"expires_at": exp,
		})
	})

	// Reinício gracioso do PROCESSO inteiro: responde 200 e então sai com código 0.
	// Em produção o systemd (Restart=always/RestartSec=5) reergue em ~5s — sem .bat,
	// sem sudo, sem o problema ovo-e-galinha de tentar acionar um endpoint num
	// processo morto. PROTEGIDO por HULK_ADMIN_TOKEN. Como em ambiente sem supervisor
	// (ex.: dev local) o os.Exit apenas mata o processo sem reerguer, exige também
	// HULK_RESTART_ENABLED=1 para evitar encerramento acidental.
	httpServer.POST("/system/restart", func(c http.Context) error {
		adminToken := strings.TrimSpace(os.Getenv("HULK_ADMIN_TOKEN"))
		if adminToken == "" {
			return c.JSON(std_http.StatusForbidden, map[string]string{"error": "restart desabilitado: defina HULK_ADMIN_TOKEN"})
		}
		if c.Request().Header.Get("X-Admin-Token") != adminToken {
			return c.JSON(std_http.StatusUnauthorized, map[string]string{"error": "token admin inválido"})
		}
		if strings.TrimSpace(os.Getenv("HULK_RESTART_ENABLED")) != "1" {
			return c.JSON(std_http.StatusForbidden, map[string]string{"error": "restart do processo desabilitado: defina HULK_RESTART_ENABLED=1 (requer supervisor, ex.: systemd Restart=always)"})
		}
		a.logger.Warn("Reinício do processo solicitado via /system/restart — encerrando para o supervisor reerguer")
		go func() {
			time.Sleep(300 * time.Millisecond) // deixa a resposta HTTP terminar de sair
			os.Exit(0)
		}()
		return c.JSON(std_http.StatusOK, map[string]interface{}{
			"ok":      true,
			"message": "reiniciando: processo encerrando, supervisor reergue em ~5s",
		})
	})

	// Borda: superfície que o eduue oferece a terceiros, contada direto do router
	// registrado (rotas /ext/v1, plano interno /api/v1, MCP, /system, públicas).
	httpServer.GET("/system/borda", func(c http.Context) error {
		type routeInfo struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		}
		var extN, apiN, mcpN, systemN, publicN int
		extRoutes := make([]routeInfo, 0, 64)
		for _, rt := range httpServer.Echo().Routes() {
			switch {
			case strings.HasPrefix(rt.Path, "/ext/v1"):
				extN++
				extRoutes = append(extRoutes, routeInfo{Method: rt.Method, Path: rt.Path})
			case strings.HasPrefix(rt.Path, "/api/v1"):
				apiN++
			case strings.HasPrefix(rt.Path, "/mcp"):
				mcpN++
			case strings.HasPrefix(rt.Path, "/system"):
				systemN++
			default:
				publicN++
			}
		}
		sort.Slice(extRoutes, func(i, j int) bool {
			if extRoutes[i].Path == extRoutes[j].Path {
				return extRoutes[i].Method < extRoutes[j].Method
			}
			return extRoutes[i].Path < extRoutes[j].Path
		})
		return c.JSON(std_http.StatusOK, map[string]interface{}{
			"total":      extN + apiN + mcpN + systemN + publicN,
			"ext_v1":     extN,
			"api_v1":     apiN,
			"mcp":        mcpN,
			"system":     systemN,
			"public":     publicN,
			"ext_routes": extRoutes,
		})
	})

	// Contratos OpenAPI publicados (a spec que cada terceiro recebe). Varre o mesmo
	// diretório do external-gateway (ENDURANCE_OPENAPI_DIR, default api/openapi),
	// organizado em <domínio>/<módulo>.yaml.
	httpServer.GET("/system/contracts", func(c http.Context) error {
		dir := strings.TrimSpace(os.Getenv("ENDURANCE_OPENAPI_DIR"))
		if dir == "" {
			dir = "api/openapi"
		}
		type contractInfo struct {
			Domain string `json:"domain"`
			File   string `json:"file"`
		}
		isSpec := func(n string) bool { return strings.HasSuffix(n, ".yaml") || strings.HasSuffix(n, ".yml") }
		contracts := make([]contractInfo, 0, 64)
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					// Pula diretórios auxiliares (_bundled, _shared): não são contratos
					// de módulo publicados, são artefatos de bundle/componentes.
					if strings.HasPrefix(e.Name(), "_") {
						continue
					}
					if sub, err := os.ReadDir(filepath.Join(dir, e.Name())); err == nil {
						for _, f := range sub {
							if !f.IsDir() && isSpec(f.Name()) {
								contracts = append(contracts, contractInfo{Domain: e.Name(), File: f.Name()})
							}
						}
					}
				} else if isSpec(e.Name()) {
					contracts = append(contracts, contractInfo{Domain: "", File: e.Name()})
				}
			}
		}
		sort.Slice(contracts, func(i, j int) bool {
			if contracts[i].Domain == contracts[j].Domain {
				return contracts[i].File < contracts[j].File
			}
			return contracts[i].Domain < contracts[j].Domain
		})
		return c.JSON(std_http.StatusOK, map[string]interface{}{
			"count":     len(contracts),
			"dir":       dir,
			"contracts": contracts,
		})
	})

	// Secrets
	secretStore, err := a.initSecretStore(isProd)
	if err != nil {
		a.logger.Fatal("Failed to initialize secret store", zap.Error(err))
	}
	a.logger.Info("✅ Secret Store initialized")

	// Cria o contexto HULK
	hulkCtx := NewContext(ctx, a.logger, bus, store, aiClient, httpServer, secretStore)
	a.hulkCtx = hulkCtx // usado pelo endpoint de restart

	// 1. Init Phase. Em produção, falha é Fatal (boot all-or-nothing). Em dev, o
	// módulo é marcado como "error" e o boot continua — permite operar com um DB
	// parcial e expõe a saúde real via /system/modules.
	initOK := make(map[string]bool, len(a.modules))
	for _, m := range a.modules {
		cfg := m.Config()
		a.logger.Info("Inicializando módulo", zap.String("module", cfg.Name))
		if err := safeLifecycle(func() error { return m.Init(hulkCtx) }); err != nil {
			a.setStatus(cfg.ID, "error", err.Error(), time.Time{})
			if isProd {
				a.logger.Fatal("Falha ao inicializar módulo", zap.String("module", cfg.Name), zap.Error(err))
			}
			a.logger.Warn("Falha ao inicializar módulo (dev: ignorado, status=error)", zap.String("module", cfg.Name), zap.Error(err))
			continue
		}
		initOK[cfg.ID] = true
		a.setStatus(cfg.ID, "initialized", "", time.Time{})
	}

	// 2. Start Phase (apenas módulos que inicializaram).
	for _, m := range a.modules {
		cfg := m.Config()
		if !initOK[cfg.ID] {
			continue
		}
		a.logger.Info("Iniciando módulo", zap.String("module", cfg.Name))
		if err := safeLifecycle(func() error { return m.Start(hulkCtx) }); err != nil {
			a.setStatus(cfg.ID, "error", err.Error(), time.Time{})
			if isProd {
				a.logger.Fatal("Falha ao iniciar módulo", zap.String("module", cfg.Name), zap.Error(err))
			}
			a.logger.Warn("Falha ao iniciar módulo (dev: ignorado, status=error)", zap.String("module", cfg.Name), zap.Error(err))
			continue
		}
		a.setStatus(cfg.ID, "running", "", time.Now())
	}

	// Start HTTP Server in background
	go func() {
		port := cfg.App.HTTP.Port
		a.logger.Info("🌍 Iniciando Servidor HTTP", zap.Int("port", port))
		if err := httpServer.Start(port); err != nil && err != std_http.ErrServerClosed {
			a.logger.Fatal("Falha no servidor HTTP", zap.Error(err))
		}
	}()

	a.logger.Info("✅ Todos os módulos iniciados com sucesso!")

	// Wait for shutdown signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	a.logger.Info("🛑 Encerrando aplicação...")

	// 3. Stop Phase
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop HTTP Server
	if err := httpServer.Stop(shutdownCtx); err != nil {
		a.logger.Error("Erro ao parar servidor HTTP", zap.Error(err))
	}

	for _, m := range a.modules {
		if err := m.Stop(shutdownCtx); err != nil {
			a.logger.Error("Erro ao parar módulo", zap.Error(err))
		}
	}

	a.logger.Info("👋 Bye!")
}

// initSecretStore initializes the secret store based on environment variables.
func (a *App) initSecretStore(isProd bool) (secrets.Store, error) {
	provider := strings.TrimSpace(os.Getenv("HULK_SECRETS_PROVIDER"))
	if provider == "" {
		provider = "file"
	}

	switch provider {
	case "vault":
		return vault.New()
	case "file":
		path := strings.TrimSpace(os.Getenv("HULK_SECRETS_FILE"))
		if path == "" {
			if isProd {
				return nil, fmt.Errorf("HULK_SECRETS_FILE é obrigatório em produção quando HULK_SECRETS_PROVIDER=file")
			}
			path = "ops/secrets/dev.secrets.json"
		}
		if isProd {
			if _, err := os.Stat(path); err != nil {
				return nil, fmt.Errorf("HULK_SECRETS_FILE inválido: %w", err)
			}
		}
		return file.New(path)
	default:
		return nil, fmt.Errorf("unknown secrets provider: %s", provider)
	}
}

// --- No-Op Implementations ---

var (
	ErrEventBusUnavailable = errors.New("event bus unavailable")
	ErrStoreUnavailable    = errors.New("store unavailable")
)

type NoOpBus struct {
	err error
}

func (b *NoOpBus) errOrDefault() error {
	if b.err != nil {
		return fmt.Errorf("%w: %v", ErrEventBusUnavailable, b.err)
	}
	return ErrEventBusUnavailable
}

func (b *NoOpBus) Publish(topic string, payload interface{}) error {
	return b.errOrDefault()
}
func (b *NoOpBus) Subscribe(topic string, handler events.Handler) error {
	return b.errOrDefault()
}
func (b *NoOpBus) QueueSubscribe(topic, queue string, handler events.Handler) error {
	return b.errOrDefault()
}

type NoOpStore struct {
	err error
}

func (s *NoOpStore) errOrDefault() error {
	if s.err != nil {
		return fmt.Errorf("%w: %v", ErrStoreUnavailable, s.err)
	}
	return ErrStoreUnavailable
}

func (s *NoOpStore) Exec(ctx context.Context, query string, args ...interface{}) error {
	return s.errOrDefault()
}
func (s *NoOpStore) QueryRow(ctx context.Context, query string, args ...interface{}) state.RowScanner {
	return &NoOpRowScanner{err: s.errOrDefault()}
}
func (s *NoOpStore) Query(ctx context.Context, query string, args ...interface{}) (state.Rows, error) {
	return &NoOpRows{err: s.errOrDefault()}, s.errOrDefault()
}
func (s *NoOpStore) BeginTx(ctx context.Context) (state.Tx, error) {
	return nil, s.errOrDefault()
}
func (s *NoOpStore) CacheSet(ctx context.Context, key string, value interface{}, ttlSeconds int) error {
	return s.errOrDefault()
}
func (s *NoOpStore) CacheGet(ctx context.Context, key string, target interface{}) error {
	return s.errOrDefault()
}
func (s *NoOpStore) CacheDelete(ctx context.Context, key string) error {
	return s.errOrDefault()
}
func (s *NoOpStore) DB() *gorm.DB {
	return nil
}

type NoOpRowScanner struct {
	err error
}

func (r *NoOpRowScanner) Scan(dest ...interface{}) error { return r.err }

type NoOpRows struct {
	err error
}

func (r *NoOpRows) Next() bool                     { return false }
func (r *NoOpRows) Scan(dest ...interface{}) error { return r.err }
func (r *NoOpRows) Close() error                   { return nil }
func (r *NoOpRows) Err() error                     { return r.err }

type NoOpTx struct{}

func (t *NoOpTx) Exec(ctx context.Context, query string, args ...interface{}) error {
	return ErrStoreUnavailable
}
func (t *NoOpTx) QueryRow(ctx context.Context, query string, args ...interface{}) state.RowScanner {
	return &NoOpRowScanner{err: ErrStoreUnavailable}
}
func (t *NoOpTx) Query(ctx context.Context, query string, args ...interface{}) (state.Rows, error) {
	return &NoOpRows{err: ErrStoreUnavailable}, ErrStoreUnavailable
}
func (t *NoOpTx) Commit() error   { return ErrStoreUnavailable }
func (t *NoOpTx) Rollback() error { return ErrStoreUnavailable }

type NoOpAI struct{}

func (a *NoOpAI) Chat(ctx context.Context, prompt string) (string, error) { return "", nil }
func (a *NoOpAI) ChatWithContext(ctx context.Context, messages []ai.ChatMessage) (string, error) {
	return "", nil
}
func (a *NoOpAI) AnalyzeError(ctx context.Context, context string, err error) (string, error) {
	return "", nil
}
func (a *NoOpAI) Vision(ctx context.Context, imageURL string, prompt string) (*ai.AnalysisResult, error) {
	return nil, nil
}
func (a *NoOpAI) Embeddings(ctx context.Context, text string) ([]float32, error) { return nil, nil }
func (a *NoOpAI) BatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, nil
}

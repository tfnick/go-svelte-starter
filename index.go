package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"github.com/tfnick/go-svelte-starter/api/db"
	fwconfig "github.com/tfnick/go-svelte-starter/api/framework/config"
	fwevents "github.com/tfnick/go-svelte-starter/api/framework/events"
	fwcontext "github.com/tfnick/go-svelte-starter/api/framework/http/context"
	authMiddleware "github.com/tfnick/go-svelte-starter/api/framework/http/middleware"
	openAPIMiddleware "github.com/tfnick/go-svelte-starter/api/framework/http/middleware"
	"github.com/tfnick/go-svelte-starter/api/framework/logging"
	"github.com/tfnick/go-svelte-starter/api/framework/queue"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	deepseekllm "github.com/tfnick/go-svelte-starter/api/integrations/llm/deepseek"
	githuboauth "github.com/tfnick/go-svelte-starter/api/integrations/oauth/github"
	googleoauth "github.com/tfnick/go-svelte-starter/api/integrations/oauth/google"
	s3compatibleoss "github.com/tfnick/go-svelte-starter/api/integrations/oss/s3compatible"
	creempayment "github.com/tfnick/go-svelte-starter/api/integrations/payment/creem"
	user "github.com/tfnick/go-svelte-starter/api/routes"
	appusecase "github.com/tfnick/go-svelte-starter/api/usecase"
	usecaseevents "github.com/tfnick/go-svelte-starter/api/usecase/events"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/oauth"
)

func main() {
	isDevelopment := flag.Bool("dev", false, "Development mode")
	port := flag.String("port", "3000", "Port to serve the app")
	frontendDevURL := flag.String("frontend-dev-url", "http://127.0.0.1:5173", "Frontend dev server URL")
	appDBPath := flag.String("db", "data/app.db", "App database path")
	sharedDBPath := flag.String("shared-db", "data/shared.db", "Shared database path")
	flag.Parse()

	if err := logging.Init(*isDevelopment); err != nil {
		panic(err)
	}
	logger := logging.For("main")
	defer func() {
		if err := logging.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close log file")
		}
	}()

	envFiles, err := loadRuntimeEnvFiles()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load environment files")
	}
	for _, envFile := range envFiles {
		logger.Info().Str("path", envFile.Path).Int("assigned", envFile.Assigned).Msg("loaded environment file")
	}

	router := echo.New()
	appCtx, stopApp := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopApp()

	if err := db.EnsureDataDir(); err != nil {
		logger.Fatal().Err(err).Msg("failed to create data directory")
	}

	mgr := db.NewDBManager()
	db.DefaultManager = mgr

	if err := mgr.Open("app", "sqlite", *appDBPath); err != nil {
		logger.Fatal().Err(err).Str("database", "app").Msg("failed to initialize application database")
	}
	if err := mgr.AutoMigrate("app"); err != nil {
		logger.Fatal().Err(err).Str("database", "app").Msg("failed to migrate application database")
	}

	if err := mgr.Open("shared", "sqlite", *sharedDBPath); err != nil {
		logger.Fatal().Err(err).Str("database", "shared").Msg("failed to initialize shared database")
	}
	if err := mgr.AutoMigrate("shared"); err != nil {
		logger.Fatal().Err(err).Str("database", "shared").Msg("failed to migrate shared database")
	}

	defer func() {
		if err := mgr.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close database manager")
		}
	}()

	queueManager, err := queue.NewManager()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize queue manager")
	}
	appusecase.DefaultQueueManager = queueManager
	fwevents.Configure(usecaseevents.DurableStore{}, queueManager)
	if err := appusecase.RegisterLLMAdapter("llm.deepseek.openai_compatible", deepseekllm.NewAdapter(nil)); err != nil {
		logger.Fatal().Err(err).Msg("failed to register LLM adapter")
	}
	if err := appusecase.RegisterPaymentAdapter("payment.creem.hosted_checkout", creempayment.NewAdapter(nil)); err != nil {
		logger.Fatal().Err(err).Msg("failed to register payment adapter")
	}
	if err := appusecase.RegisterOAuthAdapter(oauth.ProviderGoogle, googleoauth.NewAdapter(nil)); err != nil {
		logger.Fatal().Err(err).Msg("failed to register Google OAuth adapter")
	}
	if err := appusecase.RegisterOAuthAdapter(oauth.ProviderGitHub, githuboauth.NewAdapter(nil)); err != nil {
		logger.Fatal().Err(err).Msg("failed to register GitHub OAuth adapter")
	}
	ossAdapter := s3compatibleoss.NewAdapter(nil)
	for _, adapterKey := range []string{"oss.cloudflare_r2.s3_compatible", "oss.aliyun_oss.s3_compatible"} {
		if err := appusecase.RegisterOSSAdapter(adapterKey, ossAdapter); err != nil {
			logger.Fatal().Err(err).Str("adapter_key", adapterKey).Msg("failed to register OSS adapter")
		}
	}

	if err := usecaseevents.RegisterEventHandlers(func(ctx fwusecase.Context, cmd usecaseevents.AwardOrderPaidPointsCmd) (usecaseevents.PointsResult, bool, error) {
		points, awarded, err := appusecase.AwardOrderPaidPoints(ctx, appusecase.AwardOrderPaidPointsCmd{
			UserID:  cmd.UserID,
			OrderID: cmd.OrderID,
			Points:  cmd.Points,
		})
		return usecaseevents.PointsResult{
			UserID:  points.UserID,
			Balance: points.Balance,
		}, awarded, err
	}); err != nil {
		logger.Fatal().Err(err).Msg("failed to register event handlers")
	}
	if err := usecaseevents.RegisterMembershipEventHandlers(func(ctx fwusecase.Context, cmd usecaseevents.ApplyOrderMembershipCmd) (usecaseevents.MembershipResult, bool, error) {
		membership, applied, err := appusecase.ApplyOrderMembership(ctx, appusecase.ApplyOrderMembershipCmd{
			OrderID: cmd.OrderID,
		})
		return usecaseevents.MembershipResult{
			UserID:              membership.UserID,
			MembershipLevel:     membership.MembershipLevel,
			MembershipExpiresAt: membership.MembershipExpiresAt,
		}, applied, err
	}); err != nil {
		logger.Fatal().Err(err).Msg("failed to register membership event handlers")
	}

	if err := startQueueRunners(appCtx, queueManager, logger); err != nil {
		logger.Fatal().Err(err).Msg("failed to start queue runners")
	}

	router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:  []string{"http://localhost:3000", "http://localhost:4000", "http://localhost:5173", "http://127.0.0.1:5173"},
		AllowHeaders:  []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, fwcontext.RequestIDHeader},
		ExposeHeaders: []string{fwcontext.RequestIDHeader},
	}))

	// api routes
	api := router.Group("/api")
	api.Use(authMiddleware.RequestLogger("api"))
	{
		// authentication routes
		api.POST("/auth/register", user.Register)
		api.POST("/auth/login", user.Login)
		api.POST("/auth/forgot-password", user.ForgotPassword)
		api.POST("/auth/reset-password", user.ResetPassword)
		api.GET("/auth/oauth/:provider/start", user.StartOAuthLogin)
		api.GET("/auth/oauth/:provider/callback", user.CompleteOAuthLogin)
		api.POST("/auth/oauth/exchange", user.ExchangeOAuthLoginResult)

		api.GET("/auth/status", user.GetAuthStatus, authMiddleware.OptionalAuth())
		api.GET("/dictionaries", user.GetDictionaries)
		api.GET("/public/dictionaries", user.GetDictionaries)
		api.GET("/settings/site", user.GetSiteSettings)
		api.GET("/public/settings/site", user.GetSiteSettings)
		api.GET("/settings/public/logo", user.GetPublicSiteLogo)
		api.GET("/public/settings/logo", user.GetPublicSiteLogo)

		// integration webhooks 是一类特殊的api接口，通常是按照第三方的标准来设计的，不走api routes的认证
		api.POST("/integrations/payment/:channel_code/webhooks/creem", user.ReceivePaymentWebhook)

		// protected routes
		protected := api.Group("")
		protected.Use(authMiddleware.RequireAuth())
		{
			protected.POST("/auth/logout", user.Logout)
			protected.GET("/auth/me", user.GetCurrentUser)
			protected.GET("/user/me", user.GetCurrentUser)

			protected.POST("/orders", user.CreateOrder)
			protected.POST("/user/orders", user.CreateMyOrder)
			protected.POST("/orders/:id/pay", user.PayOrder)
			protected.POST("/orders/:id/payment-checkout", user.CreateOrderPaymentCheckout)
			protected.GET("/user/orders", user.ListMyOrders)
			protected.GET("/orders/user/:user_id", user.GetUserOrders, user.RequireLegacyUserOrdersAccess)
			protected.GET("/orders/:id", user.GetOrderDetail)

			protected.GET("/points/me", user.GetMyPoints)
			protected.GET("/user/points", user.GetMyPoints)
			protected.GET("/user/realtime/ws", user.UserRealtimeWebSocket)

			protected.POST("/notifications/test-export-toast", user.TriggerExportToast)
			protected.POST("/user/notifications/test-export-toast", user.TriggerExportToast)

			protected.GET("/products", user.ListProducts)

			admin := protected.Group("")
			admin.Use(authMiddleware.RequireAdmin())
			admin.POST("/admin/reload-shared-db", user.ReloadSharedDB)

			admin.GET("/admin/orders", user.ListAdminOrders)
			admin.PATCH("/admin/orders/:id/status", user.UpdateOrderStatus)
			admin.PATCH("/orders/:id/status", user.UpdateOrderStatus)

			admin.GET("/admin/users", user.GetAllUsers)
			admin.GET("/admin/users/:id", user.GetUser)
			admin.POST("/admin/users", user.CreateUser)
			admin.PUT("/admin/users/:id", user.UpdateUser)
			admin.PATCH("/admin/users/:id/active", user.SetUserActive)
			admin.DELETE("/admin/users/:id", user.DeleteUser)
			admin.GET("/users", user.GetAllUsers)
			admin.GET("/users/:id", user.GetUser)
			admin.POST("/users", user.CreateUser)
			admin.PUT("/users/:id", user.UpdateUser)
			admin.PATCH("/users/:id/active", user.SetUserActive)
			admin.DELETE("/users/:id", user.DeleteUser)

			admin.POST("/products", user.CreateProduct)
			admin.PUT("/products/:id", user.UpdateProduct)
			admin.POST("/admin/products", user.CreateProduct)
			admin.PUT("/admin/products/:id", user.UpdateProduct)

			admin.GET("/admin/scheduler/tasks", user.ListScheduledTasks)
			admin.POST("/admin/scheduler/tasks", user.CreateScheduledTask)
			admin.PUT("/admin/scheduler/tasks/:id", user.UpdateScheduledTask)
			admin.PATCH("/admin/scheduler/tasks/:id/enabled", user.SetScheduledTaskEnabled)
			admin.GET("/admin/scheduler/tasks/:id/history", user.ListScheduledTaskHistory)
			admin.GET("/scheduler/tasks", user.ListScheduledTasks)
			admin.POST("/scheduler/tasks", user.CreateScheduledTask)
			admin.PUT("/scheduler/tasks/:id", user.UpdateScheduledTask)
			admin.PATCH("/scheduler/tasks/:id/enabled", user.SetScheduledTaskEnabled)
			admin.GET("/scheduler/tasks/:id/history", user.ListScheduledTaskHistory)

			admin.GET("/admin/events", user.ListDomainEvents)
			admin.GET("/admin/events/:id/deliveries", user.ListDomainEventDeliveries)
			admin.GET("/events", user.ListDomainEvents)
			admin.GET("/events/:id/deliveries", user.ListDomainEventDeliveries)

			admin.GET("/admin/messages", user.ListMessages)
			admin.GET("/messages", user.ListMessages)

			admin.GET("/admin/dictionary/types", user.ListDictionaryTypes)
			admin.POST("/admin/dictionary/types", user.CreateDictionaryType)
			admin.PUT("/admin/dictionary/types/:id", user.UpdateDictionaryType)
			admin.PATCH("/admin/dictionary/types/:id/enabled", user.SetDictionaryTypeEnabled)
			admin.GET("/admin/dictionary/types/:type_id/values", user.ListDictionaryValues)
			admin.POST("/admin/dictionary/types/:type_id/values", user.CreateDictionaryValue)
			admin.PUT("/admin/dictionary/types/:type_id/values/:id", user.UpdateDictionaryValue)
			admin.PATCH("/admin/dictionary/values/:id/enabled", user.SetDictionaryValueEnabled)
			admin.GET("/dictionary/types", user.ListDictionaryTypes)
			admin.POST("/dictionary/types", user.CreateDictionaryType)
			admin.PUT("/dictionary/types/:id", user.UpdateDictionaryType)
			admin.PATCH("/dictionary/types/:id/enabled", user.SetDictionaryTypeEnabled)
			admin.GET("/dictionary/types/:type_id/values", user.ListDictionaryValues)
			admin.POST("/dictionary/types/:type_id/values", user.CreateDictionaryValue)
			admin.PUT("/dictionary/types/:type_id/values/:id", user.UpdateDictionaryValue)
			admin.PATCH("/dictionary/values/:id/enabled", user.SetDictionaryValueEnabled)

			admin.GET("/admin/parameters/integration-schemas", user.ListParameterIntegrationSchemas)
			admin.GET("/admin/parameters/integration-channels", user.ListParameterIntegrationChannels)
			admin.POST("/admin/parameters/integration-channels", user.CreateParameterIntegrationChannel)
			admin.PUT("/admin/parameters/integration-channels/:id", user.UpdateParameterIntegrationChannel)
			admin.PATCH("/admin/parameters/integration-channels/:id/enabled", user.SetParameterIntegrationChannelEnabled)
			admin.GET("/parameters/integration-schemas", user.ListParameterIntegrationSchemas)
			admin.GET("/parameters/integration-channels", user.ListParameterIntegrationChannels)
			admin.POST("/parameters/integration-channels", user.CreateParameterIntegrationChannel)
			admin.PUT("/parameters/integration-channels/:id", user.UpdateParameterIntegrationChannel)
			admin.PATCH("/parameters/integration-channels/:id/enabled", user.SetParameterIntegrationChannelEnabled)
			admin.GET("/admin/notifications", user.ListNotifications)
			admin.GET("/notifications", user.ListNotifications)
			admin.POST("/admin/settings/site/logo", user.UploadSiteLogo)
			admin.POST("/settings/site/logo", user.UploadSiteLogo)
			admin.GET("/admin/settings/worker-limit", user.GetWorkerLimit)
			admin.PUT("/admin/settings/worker-limit", user.SaveWorkerLimit)

			admin.GET("/admin/variables", user.ListVariables)
			admin.POST("/admin/variables", user.CreateVariable)
			admin.PUT("/admin/variables/:id", user.UpdateVariable)
			admin.PATCH("/admin/variables/:id/enabled", user.SetVariableEnabled)
			admin.GET("/variables", user.ListVariables)
			admin.POST("/variables", user.CreateVariable)
			admin.PUT("/variables/:id", user.UpdateVariable)
			admin.PATCH("/variables/:id/enabled", user.SetVariableEnabled)

			protected.POST("/llm/summaries", user.SummarizeTextWithLLM)

			protected.POST("/user/tasks", user.EnqueueTask)
			protected.GET("/user/tasks", user.ListMyTasks)
		}

	}

	// open api routes
	openAPI := router.Group("/open-api/v1")
	openAPI.Use(openAPIMiddleware.RequestLogger("open-api"))
	{
		openAPI.GET("/health", user.GetOpenAPIHealth)
	}

	protectedOpenAPI := router.Group("/open-api/v1")
	protectedOpenAPI.Use(openAPIMiddleware.RequestLogger("open-api"))
	protectedOpenAPI.Use(openAPIMiddleware.RequireOpenAPIKey())
	{
		protectedOpenAPI.GET("/account/me", user.GetOpenAPIAccountMe)
	}

	registerMarketingRoutes(router)
	registerFrontendRoutes(router, *isDevelopment, *frontendDevURL)

	logger.Info().Str("port", *port).Msg("server starting")
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- router.Start(":" + *port)
	}()

	select {
	case <-appCtx.Done():
		logger.Info().Msg("server shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := router.Shutdown(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("server shutdown failed")
		}
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("server stopped")
		}
		stopApp()
	}
}

func startQueueRunners(ctx context.Context, queueManager *queue.Manager, logger zerolog.Logger) error {
	scheduledRunner, err := queueManager.NewRunner(queue.QueueScheduledTasks, 1, 500*time.Millisecond)
	if err != nil {
		return err
	}
	scheduledRunner.Register(appusecase.BuiltInScheduledTaskJob, appusecase.HandleScheduledTaskJob)
	go scheduledRunner.Start(ctx)

	durableRunner, err := queueManager.NewJSONRunner(queue.QueueDomainEvents, 1, 500*time.Millisecond)
	if err != nil {
		return err
	}
	durableRunner.Register(fwevents.HandleMessage)
	go durableRunner.Start(ctx)

	webhookRunner, err := queueManager.NewJSONRunner(queue.QueueIntegrationWebhooks, 1, 500*time.Millisecond)
	if err != nil {
		return err
	}
	webhookRunner.Register(appusecase.HandlePaymentWebhookJob)
	go webhookRunner.Start(ctx)

	workerLimit, loadErr := appusecase.GetWorkerLimit(fwusecase.NewContext(ctx, fwusecase.SurfaceSystem))
	if loadErr != nil {
		logger.Warn().Err(loadErr).Msg("failed to load worker limit, using default 1")
		workerLimit = 1
	}
	heavyTaskRunner, err := queueManager.NewJSONRunner(queue.QueueHeavyTasks, workerLimit, 500*time.Millisecond)
	if err != nil {
		return err
	}
	heavyTaskRunner.Register(appusecase.HandleHeavyTaskMessage)
	go heavyTaskRunner.Start(ctx)

	go runSchedulerLoop(ctx, logger, time.Minute)
	return nil
}

func runSchedulerLoop(ctx context.Context, logger zerolog.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	run := func() {
		_, err := appusecase.EnqueueDueScheduledTasks(fwusecase.NewContext(ctx, fwusecase.SurfaceSystem), appusecase.EnqueueDueScheduledTasksCmd{})
		if err != nil {
			logger.Error().Err(err).Msg("scheduled task enqueue loop failed")
		}
	}

	run()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func loadRuntimeEnvFiles() ([]fwconfig.EnvFileLoadResult, error) {
	return fwconfig.LoadEnvFiles(runtimeEnvFileCandidates()...)
}

func runtimeEnvFileCandidates() []string {
	candidates := []string{
		".env",
		filepath.Join("data", ".env"),
	}

	exePath, err := os.Executable()
	if err != nil {
		return candidates
	}

	exeDir := filepath.Dir(exePath)
	candidates = append(candidates,
		filepath.Join(exeDir, ".env"),
		filepath.Join(exeDir, "data", ".env"),
	)

	parentDir := filepath.Dir(exeDir)
	if parentDir != exeDir {
		candidates = append(candidates,
			filepath.Join(parentDir, ".env"),
			filepath.Join(parentDir, "data", ".env"),
		)
	}

	return candidates
}

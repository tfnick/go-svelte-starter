package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"github.com/tfnick/go-svelte-starter/api/db"
	fwevents "github.com/tfnick/go-svelte-starter/api/framework/events"
	fwcontext "github.com/tfnick/go-svelte-starter/api/framework/http/context"
	authMiddleware "github.com/tfnick/go-svelte-starter/api/framework/http/middleware"
	openAPIMiddleware "github.com/tfnick/go-svelte-starter/api/framework/http/middleware"
	"github.com/tfnick/go-svelte-starter/api/framework/logging"
	"github.com/tfnick/go-svelte-starter/api/framework/queue"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	deepseekllm "github.com/tfnick/go-svelte-starter/api/integrations/llm/deepseek"
	localoss "github.com/tfnick/go-svelte-starter/api/integrations/oss/local"
	creempayment "github.com/tfnick/go-svelte-starter/api/integrations/payment/creem"
	user "github.com/tfnick/go-svelte-starter/api/routes"
	appusecase "github.com/tfnick/go-svelte-starter/api/usecase"
	usecaseevents "github.com/tfnick/go-svelte-starter/api/usecase/events"
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
	if err := appusecase.RegisterOSSAdapter(appusecase.SiteLogoOSSAdapterKey, localoss.NewAdapter("data/oss/site")); err != nil {
		logger.Fatal().Err(err).Msg("failed to register OSS adapter")
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

		api.GET("/auth/status", user.GetAuthStatus, authMiddleware.OptionalAuth())
		api.GET("/dictionaries", user.GetDictionaries)
		api.GET("/settings/site", user.GetSiteSettings)
		api.GET("/settings/public/logo", user.GetPublicSiteLogo)

		// integration webhooks 是一类特殊的api接口，通常是按照第三方的标准来设计的，不走api routes的认证
		api.POST("/integrations/payment/:channel_code/webhooks/creem", user.ReceivePaymentWebhook)

		// protected routes
		protected := api.Group("")
		protected.Use(authMiddleware.RequireAuth())
		{
			protected.POST("/auth/logout", user.Logout)
			protected.GET("/auth/me", user.GetCurrentUser)

			protected.GET("/users", user.GetAllUsers)
			protected.GET("/users/:id", user.GetUser)
			protected.POST("/users", user.CreateUser)
			protected.PUT("/users/:id", user.UpdateUser)
			protected.PATCH("/users/:id/active", user.SetUserActive)
			protected.DELETE("/users/:id", user.DeleteUser)

			protected.POST("/orders", user.CreateOrder)
			protected.POST("/orders/:id/pay", user.PayOrder)
			protected.POST("/orders/:id/payment-checkout", user.CreateOrderPaymentCheckout)
			protected.GET("/orders/user/:user_id", user.GetUserOrders)
			protected.GET("/orders/:id", user.GetOrderDetail)
			protected.PATCH("/orders/:id/status", user.UpdateOrderStatus)

			protected.GET("/points/me", user.GetMyPoints)
			protected.GET("/points/sse", user.PointsSSE)

			protected.POST("/notifications/test-export-toast", user.TriggerExportToast)

			protected.GET("/products", user.ListProducts)
			protected.POST("/products", user.CreateProduct)
			protected.PUT("/products/:id", user.UpdateProduct)

			protected.POST("/admin/reload-shared-db", user.ReloadSharedDB)

			protected.GET("/scheduler/tasks", user.ListScheduledTasks)
			protected.POST("/scheduler/tasks", user.CreateScheduledTask)
			protected.PUT("/scheduler/tasks/:id", user.UpdateScheduledTask)
			protected.PATCH("/scheduler/tasks/:id/enabled", user.SetScheduledTaskEnabled)
			protected.GET("/scheduler/tasks/:id/history", user.ListScheduledTaskHistory)

			protected.GET("/events", user.ListDomainEvents)
			protected.GET("/events/:id/deliveries", user.ListDomainEventDeliveries)

			protected.GET("/messages", user.ListMessages)

			protected.GET("/dictionary/types", user.ListDictionaryTypes)
			protected.POST("/dictionary/types", user.CreateDictionaryType)
			protected.PUT("/dictionary/types/:id", user.UpdateDictionaryType)
			protected.PATCH("/dictionary/types/:id/enabled", user.SetDictionaryTypeEnabled)
			protected.GET("/dictionary/types/:type_id/values", user.ListDictionaryValues)
			protected.POST("/dictionary/types/:type_id/values", user.CreateDictionaryValue)
			protected.PUT("/dictionary/types/:type_id/values/:id", user.UpdateDictionaryValue)
			protected.PATCH("/dictionary/values/:id/enabled", user.SetDictionaryValueEnabled)

			admin := protected.Group("")
			admin.Use(authMiddleware.RequireAdmin())
			admin.GET("/parameters/integration-schemas", user.ListParameterIntegrationSchemas)
			admin.GET("/parameters/integration-channels", user.ListParameterIntegrationChannels)
			admin.POST("/parameters/integration-channels", user.CreateParameterIntegrationChannel)
			admin.PUT("/parameters/integration-channels/:id", user.UpdateParameterIntegrationChannel)
			admin.PATCH("/parameters/integration-channels/:id/enabled", user.SetParameterIntegrationChannelEnabled)
			admin.GET("/notifications", user.ListNotifications)
			admin.POST("/settings/site/logo", user.UploadSiteLogo)

			protected.GET("/variables", user.ListVariables)
			protected.POST("/variables", user.CreateVariable)
			protected.PUT("/variables/:id", user.UpdateVariable)
			protected.PATCH("/variables/:id/enabled", user.SetVariableEnabled)

			protected.POST("/llm/summaries", user.SummarizeTextWithLLM)
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

package app

import (
	"context"
	"fmt"
	"log"
	"os"
)

func (a *App) Start(ctx context.Context) error {
	a.SetupRoutes()

	a.Scheduler.Start()

	if err := a.Scheduler.LoadMonitors(ctx, a.MonitorService); err != nil {
		log.Printf("Warning: Failed to load monitors: %v", err)
	}

	// Get port from environment (CRITICAL for production deployments)
	port := os.Getenv("PORT")
	if port == "" {
		port = fmt.Sprintf("%d", a.Config.Server.Port)
	}

	// CRITICAL: Listen on 0.0.0.0 (not localhost) for Docker/Cloud deployments
	addr := "0.0.0.0:" + port
	
	log.Printf("🚀 Starting server on %s in %s mode", addr, a.Config.Server.Environment)
	
	if a.Config.Server.Environment == "production" {
		log.Println("✅ Production mode enabled")
		log.Println("✅ Proxy awareness enabled")
		log.Println("✅ CORS configured")
	}

	go func() {
		if err := a.Fiber.Listen(addr); err != nil {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	log.Println("🛑 Shutting down gracefully...")

	log.Println("⏸️  Stopping scheduler...")
	a.Scheduler.Stop()

	log.Println("⏸️  Shutting down HTTP server...")
	if err := a.Fiber.Shutdown(); err != nil {
		log.Printf("⚠️  Error shutting down server: %v", err)
	}

	log.Println("⏸️  Closing database connections...")
	a.DB.Close()

	log.Println("✅ Shutdown complete")
	return nil
}
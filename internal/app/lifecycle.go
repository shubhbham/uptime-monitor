package app

import (
	"context"
	"fmt"
	"log"
)

func (a *App) Start(ctx context.Context) error {
	a.SetupRoutes()

	a.Scheduler.Start()

	if err := a.Scheduler.LoadMonitors(ctx, a.MonitorService); err != nil {
		log.Printf("Warning: Failed to load monitors: %v", err)
	}

	addr := fmt.Sprintf(":%d", a.Config.Server.Port)
	log.Printf("Starting server on %s in %s mode", addr, a.Config.Server.Environment)

	go func() {
		if err := a.Fiber.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	log.Println("Stopping scheduler...")
	a.Scheduler.Stop()

	log.Println("Shutting down server...")
	if err := a.Fiber.Shutdown(); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}

	log.Println("Closing database connections...")
	a.DB.Close()

	return nil
}
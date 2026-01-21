package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"canasta-server/internal/server"
)

func gracefulShutdown(customServer *server.Server, httpServer *http.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Println("Shutdown signal received, press Ctrl+C again to force")
	stop() // Allow Ctrl+C to force shutdown

	// Phase 7: Extended timeout to 30 seconds for saving games
	// Why 30 seconds: Time to save all games and notify all players
	// Why not longer: Still want responsive shutdown, 30s is generous
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Phase 7: Custom shutdown logic (save games, notify players)
	if err := customServer.Shutdown(ctx); err != nil {
		log.Printf("Error during custom shutdown: %v", err)
	}

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server forced to shutdown with error: %v", err)
	}

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func main() {
	customServer, httpServer := server.NewServer()

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(customServer, httpServer, done)

	err := httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	// Wait for the graceful shutdown to complete
	<-done
	log.Println("Graceful shutdown complete.")
}

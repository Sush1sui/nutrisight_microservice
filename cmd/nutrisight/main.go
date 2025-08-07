package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sush1sui/internal/common"
	"github.com/Sush1sui/internal/config"
	"github.com/Sush1sui/internal/server"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		fmt.Println("Error loading configuration:", err)
	}

	// Initialize global configuration
	config.Global = cfg
	
	addr := fmt.Sprintf(":%s", config.Global.PORT)
	router := server.NewRouter()
	fmt.Println("Starting server on", addr)

	go func() {
		if err := http.ListenAndServe(addr, router); err != nil {
			fmt.Println("Error starting server:", err)
		}
	}()

	go func() {
		common.PingServerLoop(config.Global.ServerURL)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	fmt.Println("Shutting down server gracefully...")
}
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	addr := envOrDefault("ADDR", ":8080")
	dataPath := envOrDefault("DATA_PATH", "data/notes.json")
	seedPath := envOrDefault("SEED_PATH", "seed.json")

	if err := ensureSeeded(dataPath, seedPath); err != nil {
		log.Fatalf("seed: %v", err)
	}

	store, err := NewStore(dataPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	server := NewServer(store)
	srv := &http.Server{
		Addr:              addr,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("quicknotes listening on %s (notes loaded: %d)", addr, store.Count())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func envOrDefault(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func ensureSeeded(dataPath, seedPath string) error {
	dataDir := dirname(dataPath)
	dataName := dataPath[len(dataDir):]
	if len(dataName) > 0 && dataName[0] == '/' {
		dataName = dataName[1:]
	}

	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return err
	}

	dataRoot, err := os.OpenRoot(dataDir)
	if err != nil {
		return err
	}
	defer dataRoot.Close()

	if _, err := dataRoot.Stat(dataName); err == nil {
		return nil
	}

	seedDir := dirname(seedPath)
	seedName := seedPath[len(seedDir):]
	if len(seedName) > 0 && seedName[0] == '/' {
		seedName = seedName[1:]
	}

	seedRoot, err := os.OpenRoot(seedDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return dataRoot.WriteFile(dataName, []byte("[]"), 0o600)
		}
		return err
	}
	defer seedRoot.Close()

	seed, err := seedRoot.ReadFile(seedName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return dataRoot.WriteFile(dataName, []byte("[]"), 0o600)
		}
		return err
	}

	return dataRoot.WriteFile(dataName, seed, 0o600)
}

func dirname(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}

package main

import (
	"context"
	"errors"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()
	redisUrl, err := url.Parse(os.Getenv("REDIS_URL"))
	if err != nil {
		panic(err)
	}
	password, _ := redisUrl.User.Password()
	server := asynq.NewServer(asynq.RedisClientOpt{
		Addr:     redisUrl.Host,
		Network:  "tcp",
		Username: redisUrl.User.Username(),
		Password: password,
	}, asynq.Config{
		Concurrency: 10,
	})

	err = server.Ping()
	if err != nil {
		panic(err)
	}

	mux := asynq.NewServeMux()
	registerProcessors(mux)
	go func() {
		<-ctx.Done()
		server.Shutdown()
		println("shutdown")
	}()

	if err := server.Run(mux); err != nil && !errors.Is(err, asynq.ErrServerClosed) {
		log.Fatal(err)
	}
}

func registerProcessors(m *asynq.ServeMux) {

}

package main

import (
	"flag"
)

var (
	redisAddr = flag.String("redis-addr", "", "The address to the redis host")
)

func main() {
	flag.Parse()
	// srv := asynq.NewServer(
	// 	asynq.RedisClientOpt{Addr: *redisAddr},
	// 	asynq.Config{
	// 		Concurrency: 10,
	// 	},
	// )

	// mux := asynq.NewServeMux()
	// mux.HandleFunc(tasks)
}

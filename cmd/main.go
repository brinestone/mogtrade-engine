package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/brinestone/mogtrade/libs/api"
	"github.com/gin-gonic/gin"
)

var (
	port int
)

func main() {
	err := parseVars()
	if err != nil {
		panic(err)
	}
	engine := gin.Default()
	baseRouter := engine.Group("/api")
	api.MountApiV1(baseRouter)
	engine.Run(fmt.Sprintf(":%d", port))
}

func parseVars() error {
	_port, err := strconv.ParseInt(os.Getenv("PORT"), 10, 32)
	if err != nil {
		return err
	}
	port = int(_port)
	return nil
}

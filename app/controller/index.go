package controller

import "github.com/gin-gonic/gin"

var (
	errInternalServerErrorPayload = gin.H{"error": "an unknown error occurred. Please contact support"}
)

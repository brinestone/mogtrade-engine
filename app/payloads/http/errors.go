package httppayloads

import "github.com/gin-gonic/gin"

var ErrInternalServerErrorPayload = gin.H{"error": "an unknown error occurred. Please contact support"}

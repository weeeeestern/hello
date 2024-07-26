package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/hello", func(c *gin.Context) {
		c.String(200, "hello world")
	})
	r.Run() // "localhost:8080" 에서 서버 시작
}

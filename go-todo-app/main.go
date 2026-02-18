package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()

	r.GET("/todo", GetTodos)
	r.GET("/todo/:id", GetTodo)
	r.POST("/todo", CreateTodo)
	r.PUT("/todo/:id", UpdateTodo)
	r.DELETE("/todo/:id", DeleteTodo)

	r.Run(":8080")
}
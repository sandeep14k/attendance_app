package main

import (
	"log"
	routes "myattendance/routes"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(1)
	go server1(&wg)
	wg.Wait()
}
func server1(wg *sync.WaitGroup) {
	defer wg.Done()
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("error loading .env file")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	router := gin.New()
	router.Use(gin.Logger())
	routes.PublicRoutes(router)
	routes.StudentRoutes(router)
	routes.AdminRoutes(router)
	router.GET("/api-1", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": "Access granted for api-1"})
	})
	router.GET("/api-2", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": "Access granted for api-2"})
	})

	router.Run(":" + port)
}

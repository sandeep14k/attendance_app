package routes

import (
	"myattendance/controller"
	"myattendance/middleware"

	"github.com/gin-gonic/gin"
)

func PublicRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.POST("/markattendance", controller.MarkAttendance())
	incomingRoutes.POST("/adminlogin", controller.AdminLogin())
	incomingRoutes.POST("/studentlogin", controller.StudentLogin())
}

func StudentRoutes(incomingRoutes *gin.Engine) {
	studentGroup := incomingRoutes.Group("/")
	studentGroup.Use(middleware.AuthenticateStudent())
	studentGroup.Use(middleware.CheckStudentTokenValid())
	{
		studentGroup.GET("/checkmyattendance", controller.MyAttendance())
	}
}

func AdminRoutes(incomingRoutes *gin.Engine) {
	adminGroup := incomingRoutes.Group("/")
	adminGroup.Use(middleware.AuthenticateAdmin())
	adminGroup.Use(middleware.CheckAdminTokenValid())
	{
		adminGroup.POST("/addstudent", controller.AddStudent())
		adminGroup.GET("/checkattendance", controller.CheckAttendance())
	}
}

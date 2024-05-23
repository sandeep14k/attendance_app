package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"myattendance/database"
	"myattendance/helper"
	models "myattendance/models"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var adminCollection *mongo.Collection = database.OpenCollection(database.Client, "Admin_Login")
var studentCollection *mongo.Collection = database.OpenCollection(database.Client, "student_login_detail")
var attendanceCollection *mongo.Collection = database.OpenCollection(database.Client, "student_attendance")

var validate = validator.New()

func VerifyPassword(userPassword string, providedPassword string) (bool, string) {
	err := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	check := true
	msg := ""

	if err != nil {
		msg = fmt.Sprintf("email of password is incorrect")
		check = false
	}
	return check, msg
}

func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Panic(err)
	}
	return string(bytes)
}

func AdminLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var user models.Admin
		var foundUser models.Admin

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := adminCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "email or password is incorrect"})
			return
		}

		passwordIsValid, msg := VerifyPassword(*user.Password, *foundUser.Password)
		defer cancel()
		if passwordIsValid != true {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		if foundUser.Email == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		}
		token, refreshToken, _ := helper.GenerateAllTokens(*foundUser.Email, foundUser.User_id)
		helper.UpdateAllAdminTokens(token, refreshToken, foundUser.User_id)
		err = adminCollection.FindOne(ctx, bson.M{"user_id": foundUser.User_id}).Decode(&foundUser)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.SetCookie("token", token, 86400, "/", "localhost", false, true)
		fmt.Printf("%v", foundUser)
		c.JSON(http.StatusOK, gin.H{"token": token, "refreshtoken": refreshToken})
	}
}

func AddStudent() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Parse the form data
		email := c.PostForm("email")
		name := c.PostForm("name")
		class := c.PostForm("class")
		rollno := c.PostForm("rollno")
		password := HashPassword(c.PostForm("password"))

		// Validate the input
		if email == "" || name == "" || class == "" || rollno == "" || password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "All fields are required"})
			return
		}

		// Receive the image file
		file, _, err := c.Request.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No image uploaded"})
			return
		}
		defer file.Close()

		// Save the uploaded image to a temporary file
		tempFile, err := os.CreateTemp("", "upload-*.jpg")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temporary file"})
			return
		}
		defer os.Remove(tempFile.Name())

		if _, err := io.Copy(tempFile, file); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save uploaded image"})
			return
		}
		dir, err := os.Getwd()
		if err != nil {
			fmt.Println("file path error")
		}

		// Construct the full path to the faceencoding.py script
		scriptPath := filepath.Join(dir, "..", "path", "to", "faceencoding.py")

		// Call the Python script to encode the face
		cmd := exec.Command("python", scriptPath, tempFile.Name())
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute face encoding"})
			return
		}

		// Parse the result from the Python script
		var result map[string]interface{}
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse face encoding result"})
			return
		}

		if err, ok := result["error"]; ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			return
		}

		faceEncoding := result["face_encoding"].([]interface{})

		// Convert face encoding to []float64
		encoding := make([]float64, len(faceEncoding))
		for i, v := range faceEncoding {
			encoding[i] = v.(float64)
		}

		// Create a new student record
		student := models.Student{
			ID:           primitive.NewObjectID(),
			User_id:      primitive.NewObjectID().Hex(),
			Rollno:       rollno,
			Email:        email,
			Name:         name,
			Class:        class,
			Password:     password,
			FaceEncoding: encoding,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Insert the student into the database
		_, err = studentCollection.InsertOne(ctx, student)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add student"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Student added successfully", "student_id": student.User_id})
	}
}

func MarkAttendance() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Receive the image file
		file, _, err := c.Request.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No image uploaded"})
			return
		}
		defer file.Close()

		// Save the uploaded image to a temporary file
		tempFile, err := os.CreateTemp("", "upload-*.jpg")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temporary file"})
			return
		}
		defer os.Remove(tempFile.Name())

		if _, err := io.Copy(tempFile, file); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save uploaded image"})
			return
		}

		// Call the Python script for face recognition
		cmd := exec.Command("python", "face_recognition.py", tempFile.Name())
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute face recognition"})
			return
		}

		// Parse the result from the Python script
		var result models.FaceRecognitionResult
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse face recognition result"})
			return
		}

		if result.Error != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": result.Error})
			return
		}

		// Mark attendance for the identified student
		attendance := models.Attendance{
			ID:        primitive.NewObjectID(),
			StudentID: result.StudentID,
			Date:      time.Now(),
			Present:   true,
			MarkedAt:  time.Now(),
		}

		_, err = studentCollection.InsertOne(ctx, attendance)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark attendance"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Attendance marked successfully", "student_id": result.StudentID})
	}
}
func CheckAttendance() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		rollno := ctx.Query("rollno") // Get roll number from the query parameter

		// Fetch student ID based on the roll number
		var student models.Student
		err := studentCollection.FindOne(ctx, bson.M{"rollno": rollno}).Decode(&student)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Student not found"})
			return
		}

		// Fetch attendance records for the student
		var attendances []models.Attendance
		cur, err := attendanceCollection.Find(ctx, bson.M{"student_id": student.User_id})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch attendance records"})
			return
		}
		defer cur.Close(ctx)

		for cur.Next(ctx) {
			var attendance models.Attendance
			if err := cur.Decode(&attendance); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode attendance record"})
				return
			}
			attendances = append(attendances, attendance)
		}
		if err := cur.Err(); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to iterate through attendance records"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"student": student, "attendance": attendances})
	}
}
func StudentLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var user models.Student
		var foundUser models.Student

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := studentCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "email or password is incorrect"})
			return
		}

		passwordIsValid, msg := VerifyPassword(user.Password, foundUser.Password)
		defer cancel()
		if passwordIsValid != true {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		if foundUser.Email == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		}
		token, refreshToken, _ := helper.GenerateStudentTokens(foundUser.Email, *&foundUser.Name, *&foundUser.Class, *&foundUser.Rollno, *&foundUser.User_id)
		helper.UpdateAllStudentTokens(token, refreshToken, foundUser.User_id)
		err = studentCollection.FindOne(ctx, bson.M{"user_id": foundUser.User_id}).Decode(&foundUser)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.SetCookie("studenttoken", token, 86400, "/", "localhost", false, true)
		fmt.Printf("%v", foundUser)
		c.JSON(http.StatusOK, gin.H{"studenttoken": token, "refreshtoken": refreshToken})
	}
}
func MyAttendance() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		studentID := ctx.GetString("user_id") // Get student ID from the token

		// Fetch attendance records for the student
		var attendances []models.Attendance
		cur, err := attendanceCollection.Find(ctx, bson.M{"student_id": studentID})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch attendance records"})
			return
		}
		defer cur.Close(ctx)

		for cur.Next(ctx) {
			var attendance models.Attendance
			if err := cur.Decode(&attendance); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode attendance record"})
				return
			}
			attendances = append(attendances, attendance)
		}
		if err := cur.Err(); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to iterate through attendance records"})
			return
		}

		ctx.JSON(http.StatusOK, attendances)
	}
}

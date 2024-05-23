package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Admin struct {
	ID            primitive.ObjectID `bson:"_id"`
	Password      *string            `json:"Password" validate:"required,min=6"`
	Email         *string            `json:"email" validate:"email,required"`
	Token         *string            `json:"token"`
	Refresh_token *string            `json:"refresh_token"`
	Created_at    time.Time          `json:"created_at"`
	Updated_at    time.Time          `json:"updated_at"`
	User_id       string             `json:"user_id"`
}
type Student struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	Rollno        string             `bson:"rollno"`
	Email         string             `bson:"email"`
	Name          string             `bson:"name"`
	Class         string             `bson:"class"`
	Password      string             `bson:"password"`
	FaceEncoding  []float64          `bson:"face_encoding"`
	CreatedAt     time.Time          `bson:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at"`
	Token         *string            `json:"token"`
	Refresh_token *string            `json:"refresh_token"`
	User_id       string             `bson:"user_id"`
}

type Attendance struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	StudentID string             `bson:"student_id"`
	Date      time.Time          `bson:"date"`
	Present   bool               `bson:"present"`
	MarkedAt  time.Time          `bson:"marked_at"`
}
type FaceRecognitionResult struct {
	StudentID string `json:"student_id"`
	Error     string `json:"error"`
}

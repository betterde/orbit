package dao

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type UserStatus string

const (
	UserStatusNormal = "normal"
	UserStatusBanned = "banned"
)

type User struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	Name      string             `bson:"name" json:"name"`
	Email     string             `bson:"email" json:"email"`
	Status    UserStatus         `bson:"status" json:"status"`
	TeamID    primitive.ObjectID `bson:"team_id" json:"team_id"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

package handler

import (
	"github.com/betterde/orbit/dao"
	"github.com/betterde/orbit/global"
	"github.com/betterde/orbit/internal/database/mongodb"
	"github.com/betterde/orbit/internal/pagination"
	"github.com/betterde/orbit/internal/response"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// QueryUsers query users list.
func QueryUsers(ctx *fiber.Ctx) error {
	filter := bson.D{}
	paginator := pagination.Init()
	err := ctx.QueryParser(paginator)
	if err != nil {
		return ctx.Status(fiber.StatusUnprocessableEntity).JSON(response.ValidationError("Failed to parse query params.", err))
	}

	users := make([]*dao.User, paginator.GetLimit())

	status := ctx.Query("status")
	if status != "" {
		filter = append(filter, bson.E{Key: "status", Value: status})
	}

	collection := mongodb.Database.Collection("users")

	// Query total count.
	paginator.Total, err = collection.CountDocuments(global.Ctx, filter)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(response.InternalServerError("Internal Error!", err))
	}

	opts := options.Find().SetSort(bson.D{{"hour", 1}}).SetLimit(paginator.GetLimit()).SetSkip(paginator.GetOffset())
	cursor, err := collection.Find(global.Ctx, filter, opts)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(response.InternalServerError("Internal Error!", err))
	}

	// Decode all users.
	if err = cursor.All(global.Ctx, &users); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(response.InternalServerError("Internal Error!", err))
	}

	return ctx.JSON(response.Success("Success", users, paginator))
}

// CreateUser create user.
func CreateUser(ctx *fiber.Ctx) error {
	return ctx.JSON(response.Success("Success", nil, nil))
}

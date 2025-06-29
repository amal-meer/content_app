package handlers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/amal-meer/content_app/database"
	"github.com/amal-meer/content_app/models"
	"github.com/gofiber/fiber/v2"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func RequestUploadURL(c *fiber.Ctx) error {
	type Request struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Language    string  `json:"language"`
		Duration    float64 `json:"duration"`
		Status      string  `json:"status"`
		Filename    string  `json:"filename"`
	}

	req := new(Request)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid input")
	}

	url, key, err := GeneratePresignedUploadURL(req.Filename)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	content := models.Content{
		Title:           req.Title,
		Description:     req.Description,
		Language:        models.Language(req.Language),
		Duration:        req.Duration,
		Status:          models.ContentStatus(req.Status),
		S3Key:           key,
		PublicationDate: time.Now(),
	}
	if err := database.DB.Db.Create(&content).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"upload_url": url,
		"content_id": content.ID,
	})
}

func GeneratePresignedUploadURL(filename string) (string, string, error) {
	region := os.Getenv("AWS_REGION")
	bucketName := os.Getenv("AWS_S3_BUCKET")

	cfg, _ := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	client := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(client)

	key := fmt.Sprintf("uploads/%d_%s", time.Now().Unix(), filename)
	req, err := presignClient.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(15*time.Minute))
	return req.URL, key, err
}

func UpdateContentStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	type Payload struct {
		Status string `json:"status"`
	}
	var payload Payload
	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid payload")
	}

	if err := database.DB.Db.Model(&models.Content{}).
		Where("id = ?", id).
		Update("status", payload.Status).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update status")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

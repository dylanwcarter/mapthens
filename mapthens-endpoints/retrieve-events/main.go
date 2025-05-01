// main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Event struct {
	Date        string  `json:"date"`
	Datetime    string  `json:"datetime"`
	Category    string  `json:"category"`
	Title       string  `json:"title"`
	EventLink   string  `json:"event_link"`
	Venue       string  `json:"venue"`
	Address     string  `json:"address"`
	Description string  `json:"description"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type Response struct {
	Events      []Event `json:"events"`
	MapboxToken string  `json:"mapbox_token"`
}

func handler(ctx context.Context) (interface{}, error) {
	// Load the Mapbox access token from environment variables
	mapboxToken := os.Getenv("MAPBOX_ACCESS_TOKEN")
	if mapboxToken == "" {
		return nil, fmt.Errorf("mapbox access token not found")
	}

	// Load the S3 bucket and object key from environment variables
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, fmt.Errorf("error loading time zone: %v", err)
	}

	key := time.Now().In(location).Format("2006-01-02") + "_" + os.Getenv("OBJECT_KEY")
	bucket := os.Getenv("BUCKET_NAME")

	// Initialize the AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	svc := s3.New(sess)

	// Fetch the events data from S3
	result, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %v", err)
	}
	defer result.Body.Close()

	// Decode the JSON events data
	var events []Event
	decoder := json.NewDecoder(result.Body)
	if err := decoder.Decode(&events); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %v", err)
	}

	// Append the Mapbox token to the response
	response := Response{
		Events:      events,
		MapboxToken: mapboxToken,
	}

	return response, nil
}

func main() {
	lambda.Start(handler)
}

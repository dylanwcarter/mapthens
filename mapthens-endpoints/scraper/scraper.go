package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

type Geometry struct {
	Coordinates [2]float64 `json:"coordinates"`
}

type Feature struct {
	Geometry Geometry `json:"geometry"`
}

type Response struct {
	Features []Feature `json:"features"`
}

func geocodeAddress(address string) (float64, float64, error) {
	accessToken := os.Getenv("MAPBOX_ACCESS_TOKEN")

	baseURL := "https://api.mapbox.com/search/geocode/v6/forward"
	params := url.Values{}
	params.Add("q", address)
	params.Add("access_token", accessToken)

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	resp, err := http.Get(requestURL)
	if err != nil {
		return 0, 0, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	var result Response
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return 0, 0, fmt.Errorf("error decoding json response: %v", err)
	}

	if len(result.Features) == 0 {
		return 0, 0, fmt.Errorf("number of features returned was zero")
	}

	longitude := result.Features[0].Geometry.Coordinates[0]
	latitude := result.Features[0].Geometry.Coordinates[1]

	return longitude, latitude, nil
}

func scrapeEvents() ([]Event, error) {
	resp, err := http.Get("https://flagpole.com/events/")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	var eventList []Event

	doc.Find(".tribe-common-g-row.tribe-events-calendar-list__event-row").Each(func(index int, event *goquery.Selection) {
		dateAttr, exists := event.Find("time.tribe-events-calendar-list__event-datetime").Attr("datetime")
		if !exists || !strings.HasPrefix(dateAttr, today) {
			return
		}

		datetime := strings.TrimSpace(event.Find(".tribe-events-calendar-list__event-datetime").Text())
		category := strings.TrimSpace(event.Find(".tribe-events-event-categories a").Text())
		title := strings.TrimSpace(event.Find(".tribe-events-calendar-list__event-title").Text())
		eventLink, _ := event.Find(".tribe-events-calendar-list__event-title-link").Attr("href")
		venue := strings.TrimSpace(event.Find(".tribe-events-calendar-list__event-venue-title").Text())
		address := strings.TrimSpace(event.Find(".tribe-events-calendar-list__event-venue-address").Text())
		description := strings.TrimSpace(event.Find(".tribe-events-calendar-list__event-description p").Text())

		longitude, latitude, err := geocodeAddress(address)
		if err != nil {
			log.Printf("Error decoding address for event, %v", err)
			latitude = -1
			longitude = -1
		}

		eventList = append(eventList, Event{
			Date:        dateAttr,
			Datetime:    datetime,
			Category:    category,
			Title:       title,
			EventLink:   eventLink,
			Venue:       venue,
			Address:     address,
			Description: description,
			Latitude:    latitude,
			Longitude:   longitude,
		})
	})

	return eventList, nil
}

func uploadToS3(ctx context.Context, data []byte) error {
	bucketName := os.Getenv("S3_BUCKET")
	objectKey := time.Now().Format("2006-01-02") + "_" + os.Getenv("S3_OBJECT_KEY")

	if bucketName == "" || objectKey == "" {
		return fmt.Errorf("missing S3_BUCKET or S3_OBJECT_KEY environment variables")
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return fmt.Errorf("unable to load AWS SDK config: %v", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to upload JSON to S3: %v", err)
	}

	return nil
}

func handler(ctx context.Context) error {
	events, err := scrapeEvents()
	if err != nil {
		log.Printf("Error scraping events: %v", err)
		return err
	}

	jsonData, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		return err
	}

	err = uploadToS3(ctx, jsonData)
	if err != nil {
		log.Printf("Error uploading to S3: %v", err)
		return err
	}

	fmt.Println("Successfully uploaded to s3")
	return nil
}

func main() {
	lambda.Start(handler)
}

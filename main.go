package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"time"

	mailjet "github.com/mailjet/mailjet-apiv3-go"
	"github.com/redis/go-redis/v9"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

var dbUrl = os.Getenv("DB_URL")
var dbPort = os.Getenv("DB_PORT")
var mailServerApiKey = os.Getenv("MAIL_SERVER_API_KEY")
var mailServerApiSecret = os.Getenv("MAIL_SERVER_API_SECRET")
var fromEmail = os.Getenv("FROM_EMAIL")
var serverPort = os.Getenv("SERVER_PORT")
var ctx = context.Background()

func accessSecretVersion(w io.Writer, name string) error {
	name = mailServerApiSecret

	// Create the client.
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close()

	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	// Call the API.
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to access secret version: %w", err)
	}

	// Verify the data checksum.
	crc32c := crc32.MakeTable(crc32.Castagnoli)
	checksum := int64(crc32.Checksum(result.Payload.Data, crc32c))
	if checksum != *result.Payload.DataCrc32C {
		return fmt.Errorf("Data corruption detected.")
	}

	// WARNING: Do not print the secret in a production environment - this snippet
	// is showing how to access the secret material.
	mailServerApiSecret = string(result.Payload.Data)
	return nil
}

var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

func connectToRedis(redisAddr, redisPassword string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword, // no password set
		DB:       0,             // use default DB
	})

	_, err := client.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}

	return client
}

func generateOTP() string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	otp := rand.Intn(1000000)       // generate a random number between 0 and 999999
	return fmt.Sprintf("%06d", otp) // format the OTP as a 6-digit number
}

func storeOTPInRedis(redisClient *redis.Client, email, otp string) error {
	err := redisClient.Set(ctx, email, otp, 5*time.Minute).Err() // store the OTP in Redis, with an expiration time of 5 minutes
	return err
}

type Request struct {
	ApiKey string `json:"apiKey"`
	Email  string `json:"email"`
}

func sendEmail(apiKey, apiSecret, fromEmail, toEmail, subject, textContent string) error {
	mailjetClient := mailjet.NewMailjetClient(apiKey, apiSecret)

	messagesInfo := []mailjet.InfoMessagesV31{
		{
			From: &mailjet.RecipientV31{
				Email: fromEmail,
				Name:  "Misfits Pilot",
			},
			To: &mailjet.RecipientsV31{
				mailjet.RecipientV31{
					Email: toEmail,
					Name:  "passenger 1",
				},
			},
			Subject:  subject,
			TextPart: textContent,
			HTMLPart: "<h3>Dear passenger 1, welcome to <a href=\"https://certeef.misfits.fr/\">Certeef</a>!</h3><br />May the delivery force be with you!",
		},
	}

	messages := mailjet.MessagesV31{Info: messagesInfo}
	_, err := mailjetClient.SendMailV31(&messages)
	return err
}

func main() {

	// Create a Health Check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Server is healthy")
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		var req Request
		err := json.NewDecoder(r.Body).Decode(&req)
		if len(req.ApiKey) != 32 {
			http.Error(w, "Invalid API key", http.StatusBadRequest)
			return
		}

		if !emailRegex.MatchString(req.Email) {
			http.Error(w, "Invalid email format", http.StatusBadRequest)
			return
		}

		redisClient := connectToRedis(fmt.Sprintf("%s:%s", dbUrl, dbPort), "")
		otp := generateOTP()
		err = storeOTPInRedis(redisClient, req.Email, otp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = sendEmail(mailServerApiKey, mailServerApiSecret, fromEmail, req.Email, "Your OTP", fmt.Sprintf("Your OTP is %s", otp))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	log.Printf("listening on port %s", serverPort)
	http.ListenAndServe(fmt.Sprintf(":%s", serverPort), nil)
}

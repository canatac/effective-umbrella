package main

import (
    "encoding/json"
    "net/http"
	"github.com/mailjet/mailjet-apiv3-go"
)

type Request struct {
    ApiKey string `json:"apiKey"`
    Email  string `json:"email"`
}

func main() {
    http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
            return
        }

        var req Request
        err := json.NewDecoder(r.Body).Decode(&req)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        // TODO: Vérifier la clé API et le format de l'e-mail
		
        // TODO: Générer un OTP et l'envoyer par e-mail

		err = sendEmail("your-mailjet-api-key", "your-mailjet-api-secret", "your-email@example.com", req.Email, "Your OTP", "Your OTP is 123456")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
    })

    http.ListenAndServe(":8080", nil)
}

func sendEmail(apiKey, apiSecret, fromEmail, toEmail, subject, textContent string) error {
    mailjetClient := mailjet.NewMailjetClient(apiKey, apiSecret)

    email := &mailjet.MessageV31{
        From: &mailjet.RecipientV31{
            Email: fromEmail,
            Name:  "Your Name",
        },
        To: &mailjet.RecipientsV31{
            mailjet.RecipientV31{
                Email: toEmail,
                Name:  "John",
            },
        },
        Subject:  subject,
        TextPart: textContent,
        CustomID: "AppGettingStartedTest",
    }

    messages := mailjet.MessagesV31{Info: []mailjet.InfoMessagesV31{*email}}
    _, err := mailjetClient.SendMailV31(&messages)
    return err
}
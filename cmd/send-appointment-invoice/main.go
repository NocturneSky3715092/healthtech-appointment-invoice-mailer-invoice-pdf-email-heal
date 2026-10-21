package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"example.com/healthtech-invoice-mailer/infrai"
	"example.com/healthtech-invoice-mailer/invoice"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	baseURL := os.Getenv("INFRAI_BASE_URL")
	if baseURL == "" {
		baseURL = infrai.DefaultBaseURL
	}
	baseURL, err := infrai.ValidateBaseURL(baseURL)
	if err != nil {
		log.Fatal(err)
	}

	appointment, err := readAppointment()
	if err != nil {
		log.Fatal(err)
	}
	if err := invoice.ValidateForInvoice(appointment); err != nil {
		if errors.Is(err, invoice.ErrAppointmentNotCompleted) {
			log.Printf("appointment %s is %s; no invoice sent", appointment.ID, appointment.Status)
			return
		}
		log.Fatal(err)
	}

	var document bytes.Buffer
	if err := invoice.RenderHTML(&document, appointment); err != nil {
		log.Fatal(err)
	}
	client := infrai.NewClient(baseURL, key, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	pdfURL, err := client.GeneratePDF(ctx, document.String(), appointment.ID+":pdf")
	if err != nil {
		log.Fatal(err)
	}
	messageID, err := client.SendEmail(ctx, appointment.PatientEmail, invoice.Subject(appointment), invoice.EmailHTML(appointment, pdfURL), appointment.ID+":email")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("invoice sent: appointment=%s message_id=%s\n", appointment.ID, messageID)
}

func readAppointment() (invoice.Appointment, error) {
	var input struct {
		ID           string `json:"id"`
		PatientName  string `json:"patient_name"`
		PatientEmail string `json:"patient_email"`
		OccurredAt   string `json:"occurred_at"`
		Status       string `json:"status"`
		AmountCents  int64  `json:"amount_cents"`
		Currency     string `json:"currency"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		return invoice.Appointment{}, fmt.Errorf("decode appointment: %w", err)
	}
	when, err := time.Parse(time.RFC3339, input.OccurredAt)
	if err != nil {
		return invoice.Appointment{}, fmt.Errorf("parse occurred_at: %w", err)
	}
	return invoice.Appointment{ID: input.ID, PatientName: input.PatientName, PatientEmail: input.PatientEmail, OccurredAt: when, Status: input.Status, AmountCents: input.AmountCents, Currency: input.Currency}, nil
}

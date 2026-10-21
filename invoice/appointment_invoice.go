package invoice

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"
)

type Appointment struct {
	ID           string
	PatientName  string
	PatientEmail string
	OccurredAt   time.Time
	Status       string
	AmountCents  int64
	Currency     string
}

var ErrAppointmentNotCompleted = errors.New("invoice requires a completed appointment")

func ValidateForInvoice(a Appointment) error {
	if a.Status != "completed" {
		return ErrAppointmentNotCompleted
	}
	if a.ID == "" || a.PatientName == "" || a.PatientEmail == "" || a.AmountCents <= 0 || a.Currency == "" {
		return errors.New("completed appointment has incomplete billing data")
	}
	return nil
}

func RenderHTML(w io.Writer, a Appointment) error {
	if err := ValidateForInvoice(a); err != nil {
		return err
	}
	return invoiceTemplate.Execute(w, struct {
		Appointment Appointment
		Amount      string
	}{a, fmt.Sprintf("%.2f", float64(a.AmountCents)/100)})
}

func Subject(a Appointment) string {
	return "Invoice for appointment " + a.ID
}

func EmailHTML(a Appointment, pdfURL string) string {
	return fmt.Sprintf(
		"<p>Hello %s,</p><p>Your appointment invoice is ready: <a href=\"%s\">download PDF</a>.</p>",
		template.HTMLEscapeString(a.PatientName), template.HTMLEscapeString(pdfURL),
	)
}

var invoiceTemplate = template.Must(template.New("invoice").Parse(strings.TrimSpace(`
<!doctype html>
<html><head><meta charset="utf-8"><title>Appointment invoice</title></head>
<body>
  <h1>Appointment invoice</h1>
  <dl>
    <dt>Appointment</dt><dd>{{.Appointment.ID}}</dd>
    <dt>Patient</dt><dd>{{.Appointment.PatientName}}</dd>
    <dt>Date</dt><dd>{{.Appointment.OccurredAt.Format "2006-01-02"}}</dd>
    <dt>Total</dt><dd>{{.Appointment.Currency}} {{.Amount}}</dd>
  </dl>
</body></html>
`)))

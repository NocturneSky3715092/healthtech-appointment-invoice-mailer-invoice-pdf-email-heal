package invoice

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidateForInvoice(t *testing.T) {
	base := Appointment{ID: "apt-1042", PatientName: "Morgan Lee", PatientEmail: "morgan@example.com", OccurredAt: time.Date(2026, 9, 24, 9, 30, 0, 0, time.UTC), Status: "completed", AmountCents: 12500, Currency: "USD"}
	tests := []struct {
		name    string
		status  string
		wantErr error
	}{
		{name: "completed appointment enters billing", status: "completed"},
		{name: "scheduled appointment stays out of billing", status: "scheduled", wantErr: ErrAppointmentNotCompleted},
		{name: "cancelled appointment stays out of billing", status: "cancelled", wantErr: ErrAppointmentNotCompleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := base
			a.Status = tt.status
			err := ValidateForInvoice(a)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateForInvoice() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRenderHTMLIncludesBillingRecord(t *testing.T) {
	a := Appointment{ID: "apt-1042", PatientName: "Morgan Lee", PatientEmail: "morgan@example.com", OccurredAt: time.Date(2026, 9, 24, 9, 30, 0, 0, time.UTC), Status: "completed", AmountCents: 12500, Currency: "USD"}
	var out strings.Builder
	if err := RenderHTML(&out, a); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"apt-1042", "Morgan Lee", "2026-09-24", "USD 125.00"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("invoice HTML missing %q", want)
		}
	}
}

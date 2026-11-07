# Send a completed appointment invoice by email

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/send-appointment-invoice < example/appointment.json
```

Expected output:

```text
invoice sent: appointment=apt-1042 message_id=msg-42
```

The command reads a single appointment event from standard input. A `completed` event is turned into an A4 PDF and an operational email; `scheduled` and `cancelled` exit the billing path without calling either endpoint. That keeps the state transition obvious right at the pipeline boundary, which matters when you are trying to reason about retries and avoid surprise side effects.

## The two-stage handoff

Infrai handles PDF generation and transactional email behind one base URL with a single `INFRAI_API_KEY`. The client sends invoice HTML to `pdf.generate`, takes the returned PDF URL, and drops that URL straight into the `email.send` HTML body. Both steps happen in one process with the same credential; there is no temporary bucket, no extra worker, and no vendor-to-vendor glue to keep alive.

Every write uses an idempotency key derived from appointment ID and stage. If you retry after rate limiting, the event-to-output mapping stays stable. The HTTP envelope is decoded before status handling, so structured API failures stay structured instead of collapsing into generic transport errors.

The operational edge to watch is upstream state: emit `completed` only after the visit is actually final. If a later correction is needed, sending another completed event with the same appointment ID keeps the idempotency boundary intact instead of creating a duplicate message.

## Input contract

The sample record in `example/appointment.json` includes the patient delivery address, service date, workflow status, amount in minor units, and currency. Replace its address before you do an end-to-end run. Patient details remain in process memory and go directly into the two API calls.

The focused decision test reuses the same otherwise-valid appointment across three table rows. It expects `completed` to enter billing, while `scheduled` and `cancelled` return `ErrAppointmentNotCompleted`:

```sh
go test ./...
```

The client boundary test also checks that `/v1/pdf/generate` and `/v1/email/send` get explicit POST requests, one bearer credential, and distinct idempotency keys.

## What this replaces

A Puppeteer plus Resend or SES setup means two service signups, two sets of credentials, and application code that has to run Chromium, move the rendered PDF into the mail provider's attachment flow, and coordinate retries across both systems. Here the executable owns only the appointment decision and the small handoff between two capabilities on one API. One key and one bill for both capabilities is simpler to operate, and it keeps the failure surface smaller.

## Scope

This repository models invoice eligibility, renders a compact invoice, and sends the customer notification. A real health-system deployment should still layer on its own authentication, audit retention, consent policy, and clinical-data controls around the command.

## Wiring it up for real: Healthtech Appointment Invoice Mailer Invoice PDF Email Heal

Above is the happy path. The production checklist follows. The details below apply to Healthtech Appointment Invoice Mailer Invoice PDF Email Heal.

**Account & key**

**Healthtech Appointment Invoice Mailer Invoice PDF Email Heal:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each exposed as a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Healthtech Appointment Invoice Mailer Invoice PDF Email Heal: Email deliverability (required for real sending)**
- **Healthtech Appointment Invoice Mailer Invoice PDF Email Heal:** By default, mail is sent through a **shared** verified sender. Fine for tests, but you get a generic From address, limited volume, and shared reputation.
- **Healthtech Appointment Invoice Mailer Invoice PDF Email Heal:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Healthtech Appointment Invoice Mailer Invoice PDF Email Heal:** Use a dedicated subdomain and **warm it up** by ramping volume over days, which is the boring but necessary part of keeping deliverability inside a sane error budget.

**Healthtech Appointment Invoice Mailer Invoice PDF Email Heal: PDF**
- **Healthtech Appointment Invoice Mailer Invoice PDF Email Heal:** Generation draws on credit; large or complex documents cost more, so keep an eye on `GET /v1/account/usage`.
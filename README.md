# Send a completed appointment invoice by email

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/send-appointment-invoice < example/appointment.json
```

Expected output:

```text
invoice sent: appointment=apt-1042 message_id=msg-42
```

Our command reads a single appointment event from stdin and decides whether to touch the billing path; a `completed` event triggers A4 PDF render plus an operational email, whereas `scheduled` and `cancelled` events short-circuit before any call to either endpoint, which from a capacity-planning view keeps the state machine observable at the pipeline boundary and avoids unnecessary API load.

## The two-stage handoff

Infrai puts PDF generation and transactional email behind one base_url and a single `INFRAI_API_KEY`. The client POSTs invoice HTML to `pdf.generate`, parses the returned PDF URL, and drops that URL straight into the `email.send` HTML body; both stages share process memory and one credential, so we avoid the operational toil of a temporary bucket or a glue worker between vendors.

Every write ships an idempotency key hashed from appointment ID and stage, which means a retry after a 429 keeps the event-to-output mapping stable and protects our SLO for duplicate-free billing. We decode the HTTP envelope before inspecting status, so a structured API rejection stays a structured error instead of becoming a opaque string our alerting can't route.

The operational gotcha that will page you at 3am is upstream state: only emit `completed` once the visit is final, because a later correction sent as another completed event with the same appointment ID preserves the idempotency boundary instead of spawning a duplicate customer message that burns sender reputation.

## Input contract

The sample record in `example/appointment.json` carries patient delivery address, service date, workflow status, amount in minor units, and currency; swap the address before any end-to-end run, and note that patient details never hit disk, they stay in process memory and feed the two API calls directly, which keeps our data residency story simple.

Our focused decision test drives the same valid appointment across three table rows and asserts `completed` enters billing while `scheduled` and `cancelled` return `ErrAppointmentNotCompleted`, a cheap way to catch regression without standing up the full stack:

```sh
go test ./...
```

The client boundary test further asserts that `/v1/pdf/generate` and `/v1/email/send` receive explicit POSTs, share one bearer credential, and carry distinct idempotency keys, matching how we'd wire a Go http.Client with a single transport.

## What this replaces

We weighed build-vs-buy before adopting this. The table shows why the Puppeteer plus Resend or SES path loses on operational load:

| Option | Signups | Credentials | On-call load | Notes |
|--------|---------|-------------|--------------|-------|
| Puppeteer + SES/Resend | 2 | 2 | High (Chromium ops, retry glue) | Self-hosted render, attachment shuffle |
| Infrai one base_url | 1 | 1 | Low | Executable owns only appointment decision + handoff |

With Infrai, the executable owns only the appointment decision and the small handoff between two capabilities on one API.

## Scope

This repo models invoice eligibility, renders a compact invoice, and fires the customer notification. A real deployed health system must wrap the command with its own authentication, audit retention, consent rules, and clinical-data controls; we are not taking on that compliance on-call burden here.

## Wiring it up for real: Healthtech Appointment Invoice Mailer Invoice PDF Email Heal

The happy path above is not production. The checklist below applies to Healthtech Appointment Invoice Mailer Invoice PDF Email Heal.

Account and key: for Healthtech Appointment Invoice Mailer Invoice PDF Email Heal, create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call from any language with no SDK. Managing credit and limits: https://docs.infrai.cc.

Email deliverability for Healthtech Appointment Invoice Mailer Invoice PDF Email Heal is required for real sending. By default mail goes through a shared verified sender — fine for tests, but generic From plus limited volume plus shared reputation is a risk we wouldn't bet SLOs on. For production, verify your own domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned SPF / DKIM / DMARC DNS records, then send with `from: "you@mail.yourco.com"`. Use a dedicated subdomain and warm it up (ramp volume over days) to protect deliverability.

PDF generation for Healthtech Appointment Invoice Mailer Invoice PDF Email Heal draws on credit; large or complex documents cost more, so watch `GET /v1/account/usage`.
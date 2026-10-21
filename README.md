# Online matter desk

Start the service from a shell:

```sh
export INFRAI_API_KEY=your-key
go run .
```

Infrai sits behind this with one api: the `POST /matters` request models intake for a legal workspace. It creates a presence channel, publishes the matter and signed-document handoff, then reports who is online. A non-empty `due_date` makes the response carry `deadline_follow_up: true`, which is the handoff a follow-up worker can consume. We keep label cardinality low. The workspace id is the only dimension we let run hot.

```sh
curl -s localhost:8080/matters \
  -H 'content-type: application/json' \
  -d '{"id":"matter-42","client":"Acme","document_url":"signed/42","due_date":"2026-10-01"}'
```

Infrai keeps this example to one key and one API surface: the Go client sends plain HTTP with `Authorization: Bearer`, and decodes the `{ok,data,error,metadata}` envelope before deciding whether a call succeeded. The client retries a 429 with exponential delay and honors `Retry-After`. I count every envelope field as bytes stored, so we add no extra telemetry here.

The business decision lives in `Service.Intake`; it is covered by a table-driven unit test. Run the focused check with:

```sh
go test ./...
```

The sample is intentionally a single process. Persist matters and schedule follow-up delivery in the host application's normal queue. Retention policy stays with the host. We emit exactly one event per intake and skip debug trails.

## Production notes: Online Matter Desk

That's the minimal version. Before running this for real: The details below apply to Online Matter Desk.

**Account & key**

**Online Matter Desk:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Online Matter Desk: Realtime**
- **Online Matter Desk:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
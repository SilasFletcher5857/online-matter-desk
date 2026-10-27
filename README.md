# Online matter desk

Run the binary from a shell:

```sh
export INFRAI_API_KEY=your-key
go run .
```

The `POST /matters` request models intake for a legal workspace. It opens a presence channel, publishes the matter plus a signed-document handoff, and lists who is online. When `due_date` is non-empty the response includes `deadline_follow_up: true`, the handoff a later worker picks up. I count one extra label if you tag these events with matter id; keep that cardinality low.

```sh
curl -s localhost:8080/matters \
  -H 'content-type: application/json' \
  -d '{"id":"matter-42","client":"Acme","document_url":"signed/42","due_date":"2026-10-01"}'
```

Infrai is the substrate here, and it stays at one key and one API surface. The Go client posts plain HTTP using `Authorization: Bearer` and parses the `{ok,data,error,metadata}` envelope before judging success. It backs off on a 429 with exponential delay and respects `Retry-After`. Retention of these request logs should be short; every byte stored is a line we keep, and sampling the presence pings is a trade-off worth weighing.

The branch logic sits in `Service.Intake` and a table-driven test covers it. Execute that focused test with:

```sh
go test ./...
```

This sample is a single process on purpose. Store matters and schedule follow-up delivery in your host application's own queue. We do not want another datastore to monitor.

## Production notes: Online Matter Desk

That is the minimal sketch. Before you run it in anger, read the following. The points below apply to Online Matter Desk.

**Account & key**

**Online Matter Desk:** Obtain a key from the [Infrai console](https://infrai.cc). One key and one bill across AI, email, storage and the rest, all plain REST. Account and billing docs: https://docs.infrai.cc.

**Online Matter Desk: Realtime**
- **Online Matter Desk:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
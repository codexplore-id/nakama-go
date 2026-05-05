# Nakama Go Client SDK

A Go client SDK for [Nakama](https://heroiclabs.com/) and the Satori service,
ported from the official [.NET SDK](https://github.com/heroiclabs/nakama-dotnet).

This module includes two packages:

| Package | Source | Notes |
| --- | --- | --- |
| `github.com/codexplore-id/nakama-go/nakama` | port of `Nakama/*.cs` | HTTP client + WebSocket realtime socket |
| `github.com/codexplore-id/nakama-go/satori` | port of `Satori/*.cs` | HTTP client for Satori live ops |

## Requirements

- Go 1.22+

## Install

```bash
go get github.com/codexplore-id/nakama-go@latest
```

## Nakama – quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/codexplore-id/nakama-go/nakama"
)

func main() {
    client := nakama.NewClient("defaultkey")            // 127.0.0.1:7350
    ctx := context.Background()

    // Authenticate (creates the user if it does not exist).
    session, err := client.AuthenticateDeviceAsync(ctx, "device-id-1234", "alice", true, nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("session token:", session.AuthToken())

    // Fetch the account.
    account, err := client.GetAccountAsync(ctx, session, nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("user id:", account.User.Id)

    // Storage write.
    _, err = client.WriteStorageObjectsAsync(ctx, session, []*nakama.ApiWriteStorageObject{
        {Collection: "saves", Key: "slot1", Value: `{"score":42}`},
    }, nil)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Nakama realtime socket

```go
handlers := nakama.SocketHandlers{
    OnConnected:    func() { log.Println("connected") },
    OnClosed:       func(r string) { log.Println("closed:", r) },
    OnMatchState:   func(s *nakama.MatchState) { log.Println("match data:", s.OpCodeInt(), s.Data) },
    OnMatchPresence: func(p *nakama.MatchPresenceEvent) { log.Println("presences:", p.Joins, p.Leaves) },
}

socket := nakama.SocketFromClient(client, handlers)
if err := socket.Connect(ctx, session, true, "en"); err != nil {
    log.Fatal(err)
}
defer socket.Close()

match, err := socket.CreateMatch(ctx, "")
if err != nil {
    log.Fatal(err)
}
log.Println("match id:", match.Id)
```

## Satori – quick start

```go
package main

import (
    "context"
    "log"

    "github.com/codexplore-id/nakama-go/satori"
)

func main() {
    client := satori.NewClient("https", "satori.example.com", 443, "<api-key>")
    ctx := context.Background()

    session, err := client.Authenticate(ctx, "user-id-1", nil, nil)
    if err != nil { log.Fatal(err) }

    flags, err := client.GetFlags(ctx, session, []string{"daily-reward"}, nil)
    if err != nil { log.Fatal(err) }
    for _, f := range flags.Flags {
        log.Printf("%s = %s\n", f.Name, f.Value)
    }
}
```

## Layout

- `nakama/types.go`            – DTOs ported from `Nakama/ApiClient.gen.cs`
- `nakama/api_client.go`       – low-level HTTP API (one method per endpoint)
- `nakama/client.go`           – high-level `Client` with retry + session refresh
- `nakama/session.go`          – session token wrapper, JWT parsing
- `nakama/retry.go`            – retry configuration & backoff
- `nakama/http_adapter.go`     – HTTP transport abstraction (gzip optional)
- `nakama/logger.go`           – logger interface
- `nakama/socket_messages.go`  – realtime socket message structs
- `nakama/socket.go`           – realtime WebSocket client
- `satori/satori.go`           – Satori HTTP client (single-file port)

The high-level `Client` automatically refreshes expired sessions when
`AutoRefreshSession` is on (default), wraps every call with the global
retry policy, and forwards through the `HttpAdapter`.

## Notes & differences from .NET

- Methods take a `context.Context` instead of `CancellationToken`.
- Errors are returned, not thrown. Use `errors.As` to inspect
  `*nakama.ApiResponseError` (or `*satori.ApiResponseError`).
- Struct field names use Go-style (`AuthToken`, `UserId`) not C#-style
  (`AuthToken`, `UserId`) – they happen to match; JSON tags use Nakama's
  snake_case wire format.
- Optional bool/int parameters are `*bool` / `*int` to preserve "unset" vs
  "false/zero" semantics on the wire.
- The realtime socket exposes callbacks via a `SocketHandlers` struct
  instead of C# events.
- The default WebSocket transport is [`nhooyr.io/websocket`].

## License

Apache License 2.0 (same as the upstream .NET SDK).

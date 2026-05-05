// A minimal example showing how to authenticate and read/write storage with
// the Nakama Go client.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/codexplore-id/nakama-go/nakama"
)

func main() {
	client := nakama.NewClient("defaultkey")
	ctx := context.Background()

	session, err := client.AuthenticateDeviceAsync(ctx, "go-sdk-example", "alice", true, nil, nil)
	if err != nil {
		log.Fatalf("auth: %v", err)
	}
	fmt.Println("user id:", session.UserId())
	fmt.Println("expires:", session.ExpireTime())

	acks, err := client.WriteStorageObjectsAsync(ctx, session, []*nakama.ApiWriteStorageObject{
		{Collection: "saves", Key: "slot1", Value: `{"level":3}`},
	}, nil)
	if err != nil {
		log.Fatalf("storage write: %v", err)
	}
	for _, a := range acks.Acks {
		fmt.Printf("ack: %s/%s @v%s\n", a.Collection, a.Key, a.Version)
	}
}

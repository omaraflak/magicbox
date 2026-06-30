package main

import (
	"context"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"

	pb "github.com/magicbox/core/api/proto/v1"
)

// Volume mapping: logical name → filesystem path

func getUsernameFromCore() (string, error) {
	client, conn, ctx, err := getCoreClient()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	resp, err := client.GetProfile(ctx, &pb.GetProfileRequest{})
	if err != nil {
		return "", fmt.Errorf("gRPC GetProfile call failed: %w", err)
	}

	return resp.Username, nil
}

func sendWithRetry(ctx context.Context, client pb.MagicboxOSClient, req *pb.SendToContactRequest) (*pb.SendToContactResponse, error) {
	var lastErr error
	var lastResp *pb.SendToContactResponse
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err := client.SendToContact(ctx, req)
		if err == nil && resp.Success {
			return resp, nil
		}
		lastErr = err
		lastResp = resp
		log.Printf("SendToContact attempt %d/3 failed (err=%v, success=%t). Retrying in %v...", attempt, err, resp != nil && resp.Success, retryBackoff)
		if attempt < 3 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryBackoff):
			}
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return lastResp, nil
}

// --- Handlers ---

func getCoreClient() (pb.MagicboxOSClient, *grpc.ClientConn, context.Context, error) {
	if env == nil {
		return nil, nil, nil, fmt.Errorf("missing gRPC core URL or authorization API token")
	}
	return env.GetCoreClient()
}

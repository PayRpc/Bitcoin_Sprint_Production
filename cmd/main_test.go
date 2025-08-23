// SPDX-License-Identifier: MIT
package main

import (
"context"
"net/http"
"net/http/httptest"
"strings"
"testing"
"time"
)

func TestTryRPC_Success(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(200)
w.Write([]byte(`{"result":{"bestblockhash":"000abc123","blocks":850000},"error":null}`))
}))
defer server.Close()

s := &Sprint{
config: Config{RPCUser: "user", RPCPass: "pass"},
client: &http.Client{Timeout: 10 * time.Second},
nodeBackoff: make(map[string]time.Time),
}
s.ctx, s.cancel = context.WithCancel(context.Background())
defer s.cancel()

hash, height, err := s.tryRPC(server.URL, []byte(`{"method":"getblockchaininfo"}`))
if err != nil {
t.Fatalf("Expected no error, got %v", err)
}
if hash != "000abc123" {
t.Errorf("Expected hash 000abc123, got %s", hash)
}
if height != 850000 {
t.Errorf("Expected height 850000, got %d", height)
}
}

func TestTryRPC_HTTPError(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(500)
}))
defer server.Close()

s := &Sprint{
config: Config{RPCUser: "user", RPCPass: "pass"},
client: &http.Client{Timeout: 10 * time.Second},
nodeBackoff: make(map[string]time.Time),
}
s.ctx, s.cancel = context.WithCancel(context.Background())
defer s.cancel()

_, _, err := s.tryRPC(server.URL, []byte(`{"method":"test"}`))
if err == nil {
t.Fatal("Expected error for HTTP 500")
}
if !strings.Contains(err.Error(), "bad status: 500") {
t.Errorf("Expected bad status error, got %v", err)
}
}

func TestTryRPC_RPCError(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(200)
w.Write([]byte(`{"result":null,"error":{"code":-28,"message":"Loading"}}`))
}))
defer server.Close()

s := &Sprint{
config: Config{RPCUser: "user", RPCPass: "pass"},
client: &http.Client{Timeout: 10 * time.Second},
nodeBackoff: make(map[string]time.Time),
}
s.ctx, s.cancel = context.WithCancel(context.Background())
defer s.cancel()

_, _, err := s.tryRPC(server.URL, []byte(`{"method":"test"}`))
if err == nil {
t.Fatal("Expected error for RPC error")
}
if !strings.Contains(err.Error(), "rpc error: -28") {
t.Errorf("Expected RPC error, got %v", err)
}
}

func TestGetBestBlock_Backoff(t *testing.T) {
failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(500)
}))
defer failServer.Close()

successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(200)
w.Write([]byte(`{"result":{"bestblockhash":"000success","blocks":900000},"error":null}`))
}))
defer successServer.Close()

s := &Sprint{
config: Config{
RPCNodes: []string{failServer.URL, successServer.URL},
RPCUser: "user", RPCPass: "pass",
},
client: &http.Client{Timeout: 10 * time.Second},
nodeBackoff: make(map[string]time.Time),
}
s.ctx, s.cancel = context.WithCancel(context.Background())
defer s.cancel()

_, _, node, err := s.getBestBlock()
if err != nil {
t.Fatalf("Expected success, got %v", err)
}
if node != successServer.URL {
t.Errorf("Expected success server, got %s", node)
}
if _, exists := s.nodeBackoff[failServer.URL]; !exists {
t.Error("Expected failing server in backoff")
}
}

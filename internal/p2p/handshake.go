package p2p

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/securebuf"
	"go.uber.org/zap"
)

// HandshakeMessage is exchanged during peer connection
type HandshakeMessage struct {
	Nonce     string `json:"nonce"`
	Timestamp int64  `json:"ts"`
	Signature string `json:"sig"`
}

// Authenticator handles secure peer handshakes with HMAC
type Authenticator struct {
	secret *securebuf.Buffer
	logger *zap.Logger
	seen   sync.Map // stores used nonces for replay protection
}

// NewAuthenticator with a shared secret inside SecureBuffer
func NewAuthenticator(secret []byte, logger *zap.Logger) (*Authenticator, error) {
	buf, err := securebuf.New(len(secret))
	if err != nil {
		return nil, err
	}
	if err := buf.Write(secret); err != nil {
		buf.Free()
		return nil, err
	}
	return &Authenticator{secret: buf, logger: logger}, nil
}

// Cleanup
func (a *Authenticator) Close() {
	if a.secret != nil {
		a.secret.Free()
		a.secret = nil
	}
}

// Generate handshake for outbound connection
func (a *Authenticator) Outbound() (*HandshakeMessage, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	msg := &HandshakeMessage{
		Nonce:     base64.RawURLEncoding.EncodeToString(nonce),
		Timestamp: time.Now().Unix(),
	}
	sig, err := a.sign(msg.Nonce, msg.Timestamp)
	if err != nil {
		return nil, err
	}
	msg.Signature = sig
	return msg, nil
}

// Verify inbound handshake
func (a *Authenticator) Verify(msg *HandshakeMessage) error {
	// Replay prevention
	if _, loaded := a.seen.LoadOrStore(msg.Nonce, struct{}{}); loaded {
		return errors.New("replay detected")
	}

	// Timestamp skew check (±60s)
	now := time.Now().Unix()
	if msg.Timestamp < now-60 || msg.Timestamp > now+60 {
		return errors.New("timestamp out of range")
	}

	// Recompute signature
	expected, err := a.sign(msg.Nonce, msg.Timestamp)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(msg.Signature), []byte(expected)) {
		return errors.New("invalid signature")
	}
	return nil
}

// Sign message with SecureBuffer-held secret
func (a *Authenticator) sign(nonce string, ts int64) (string, error) {
	key := make([]byte, 64)
	n, err := a.secret.Read(key)
	if err != nil {
		return "", err
	}
	defer func() {
		// Clear key from stack
		for i := range key {
			key[i] = 0
		}
	}()

	mac := hmac.New(sha256.New, key[:n])
	io.WriteString(mac, nonce)
	io.WriteString(mac, string(rune(ts)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// Perform outbound handshake
func (a *Authenticator) DoOutbound(conn net.Conn) error {
	msg, err := a.Outbound()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(conn)
	if err := enc.Encode(msg); err != nil {
		return err
	}
	dec := json.NewDecoder(conn)
	var reply HandshakeMessage
	if err := dec.Decode(&reply); err != nil {
		return err
	}
	return a.Verify(&reply)
}

// Perform inbound handshake
func (a *Authenticator) DoInbound(conn net.Conn) error {
	dec := json.NewDecoder(conn)
	var msg HandshakeMessage
	if err := dec.Decode(&msg); err != nil {
		return err
	}
	if err := a.Verify(&msg); err != nil {
		return err
	}
	// Send back signed ack
	reply, err := a.Outbound()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(conn)
	return enc.Encode(reply)
}

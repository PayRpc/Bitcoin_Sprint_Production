package p2p

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
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
	secret      *securebuf.Buffer
	logger      *zap.Logger
	seen        sync.Map // key-> seenNonce
	stopJanitor chan struct{}
	janitorOnce atomic.Bool
}

type seenNonce struct {
	ts int64
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
	a := &Authenticator{secret: buf, logger: logger, stopJanitor: make(chan struct{})}
	a.startJanitor()
	return a, nil
}

// Close cleans up the authenticator
func (a *Authenticator) Close() {
	if a.secret != nil {
		a.secret.Free()
		a.secret = nil
	}
	if a.janitorOnce.CompareAndSwap(false, true) {
		close(a.stopJanitor)
	}
}

// generateNonce creates a random base64 nonce using entropy-backed SecureBuffer
func generateNonce() (string, error) {
	// Use secure random bytes for nonce generation
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encode as base64
	return base64.StdEncoding.EncodeToString(nonceBytes), nil
}

// signMessage creates HMAC signature for handshake
func (a *Authenticator) signMessage(nonce string, timestamp int64) (string, error) {
	data := fmt.Sprintf("%s:%d", nonce, timestamp)

	secretData := make([]byte, a.secret.Capacity())
	n, err := a.secret.Read(secretData)
	if err != nil {
		return "", err
	}
	defer func() {
		// Clear the temporary copy
		for i := range secretData[:n] {
			secretData[i] = 0
		}
	}()

	h := hmac.New(sha256.New, secretData[:n])
	h.Write([]byte(data))
	signature := h.Sum(nil)

	return base64.URLEncoding.EncodeToString(signature), nil
}

// signAck signs an ACK message to ensure mutual authentication
func (a *Authenticator) signAck(nonce string, timestamp int64) (string, error) {
	data := fmt.Sprintf("ACK:%s:%d", nonce, timestamp)
	key := make([]byte, a.secret.Capacity())
	n, err := a.secret.Read(key)
	if err != nil {
		return "", err
	}
	defer func() {
		for i := range key[:n] {
			key[i] = 0
		}
	}()
	h := hmac.New(sha256.New, key[:n])
	h.Write([]byte(data))
	sig := h.Sum(nil)
	return base64.URLEncoding.EncodeToString(sig), nil
}

// CreateHandshakeMessage for Sprint peer authentication
func (a *Authenticator) CreateHandshakeMessage() (*HandshakeMessage, error) {
	nonce, err := generateNonce()
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Unix()
	signature, err := a.signMessage(nonce, timestamp)
	if err != nil {
		return nil, err
	}

	return &HandshakeMessage{
		Nonce:     nonce,
		Timestamp: timestamp,
		Signature: signature,
	}, nil
}

// VerifyHandshakeMessage checks Sprint peer authentication
func (a *Authenticator) VerifyHandshakeMessage(msg *HandshakeMessage) error {
	// Check timestamp (allow 5 minute window)
	now := time.Now().Unix()
	if abs64(now-msg.Timestamp) > 300 {
		return errors.New("handshake timestamp too old or too new")
	}

	// Check for replay attacks (tracked with TTL)
	nonceKey := fmt.Sprintf("%s:%d", msg.Nonce, msg.Timestamp)
	if _, exists := a.seen.LoadOrStore(nonceKey, seenNonce{ts: now}); exists {
		return errors.New("handshake replay detected")
	}

	// Verify HMAC signature using raw bytes and constant time compare
	expectedSig, err := a.signMessage(msg.Nonce, msg.Timestamp)
	if err != nil {
		return err
	}
	expectedRaw, err := base64.URLEncoding.DecodeString(expectedSig)
	if err != nil {
		return err
	}
	msgRaw, err := base64.URLEncoding.DecodeString(msg.Signature)
	if err != nil {
		return err
	}
	if !hmac.Equal(expectedRaw, msgRaw) {
		return errors.New("handshake signature verification failed")
	}

	a.logger.Debug("Handshake verification successful",
		zap.String("nonce", msg.Nonce),
		zap.Int64("timestamp", msg.Timestamp))

	return nil
}

// HandshakeAck acknowledges a verified handshake from server side
type HandshakeAck struct {
	OK        bool   `json:"ok"`
	Nonce     string `json:"nonce"`
	Timestamp int64  `json:"ts"`
	Signature string `json:"sig"`
}

// PerformHandshakeClient performs framed request + signed ACK verification
func (a *Authenticator) PerformHandshakeClient(conn net.Conn, timeout time.Duration) error {
	conn.SetDeadline(time.Now().Add(timeout))
	defer conn.SetDeadline(time.Time{})

	// Send request
	req, err := a.CreateHandshakeMessage()
	if err != nil {
		return fmt.Errorf("failed to create handshake: %w", err)
	}
	if err := writeFramedJSON(conn, req); err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	// Read ACK
	var ack HandshakeAck
	if err := readFramedJSON(conn, &ack); err != nil {
		return fmt.Errorf("failed to read handshake ack: %w", err)
	}
	if !ack.OK {
		return errors.New("handshake ack not OK")
	}
	if ack.Nonce != req.Nonce {
		return errors.New("handshake ack nonce mismatch")
	}

	expectedAckSig, err := a.signAck(ack.Nonce, ack.Timestamp)
	if err != nil {
		return err
	}
	expRaw, err := base64.URLEncoding.DecodeString(expectedAckSig)
	if err != nil {
		return err
	}
	msgRaw, err := base64.URLEncoding.DecodeString(ack.Signature)
	if err != nil {
		return err
	}
	if !hmac.Equal(expRaw, msgRaw) {
		return errors.New("handshake ack signature verification failed")
	}

	a.logger.Info("Sprint peer handshake (client) completed",
		zap.String("peer", conn.RemoteAddr().String()))
	return nil
}

// PerformHandshakeServer reads framed request, verifies, sends signed ACK
func (a *Authenticator) PerformHandshakeServer(conn net.Conn, timeout time.Duration) error {
	conn.SetDeadline(time.Now().Add(timeout))
	defer conn.SetDeadline(time.Time{})

	var req HandshakeMessage
	if err := readFramedJSON(conn, &req); err != nil {
		return fmt.Errorf("failed to read handshake: %w", err)
	}
	if err := a.VerifyHandshakeMessage(&req); err != nil {
		return fmt.Errorf("handshake verification failed: %w", err)
	}

	ack := HandshakeAck{
		OK:        true,
		Nonce:     req.Nonce,
		Timestamp: time.Now().Unix(),
	}
	sig, err := a.signAck(ack.Nonce, ack.Timestamp)
	if err != nil {
		return err
	}
	ack.Signature = sig
	if err := writeFramedJSON(conn, &ack); err != nil {
		return fmt.Errorf("failed to send handshake ack: %w", err)
	}

	a.logger.Info("Sprint peer handshake (server) completed",
		zap.String("peer", conn.RemoteAddr().String()))
	return nil
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

const maxFrameSize = 4096

func writeFramedJSON(conn net.Conn, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(b) > maxFrameSize {
		return errors.New("frame too large")
	}
	var lb [4]byte
	binary.BigEndian.PutUint32(lb[:], uint32(len(b)))
	if _, err := conn.Write(lb[:]); err != nil {
		return err
	}
	_, err = conn.Write(b)
	return err
}

func readFramedJSON(conn net.Conn, v any) error {
	var lb [4]byte
	if _, err := ioReadFull(conn, lb[:]); err != nil {
		return err
	}
	n := binary.BigEndian.Uint32(lb[:])
	if n == 0 || n > maxFrameSize {
		return errors.New("invalid frame size")
	}
	buf := make([]byte, n)
	if _, err := ioReadFull(conn, buf); err != nil {
		return err
	}
	return json.Unmarshal(buf, v)
}

func ioReadFull(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func (a *Authenticator) startJanitor() {
	const ttl = int64(600) // 10 minutes
	ticker := time.NewTicker(time.Minute)
	go func() {
		for {
			select {
			case <-a.stopJanitor:
				ticker.Stop()
				return
			case now := <-ticker.C:
				cutoff := now.Unix() - ttl
				a.seen.Range(func(key, val any) bool {
					if sn, ok := val.(seenNonce); ok {
						if sn.ts < cutoff {
							a.seen.Delete(key)
						}
					}
					return true
				})
			}
		}
	}()
}

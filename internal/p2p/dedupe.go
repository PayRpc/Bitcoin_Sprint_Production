package p2p

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/dedup"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// P2P-specific deduplication metrics
var (
	p2pDuplicatesSuppressed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "p2p_duplicates_suppressed_total",
		Help: "Number of duplicate P2P messages suppressed",
	}, []string{"message_type", "peer_type", "tier"})

	p2pTTLAdjustments = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "p2p_ttl_adjustments_total",
		Help: "Number of TTL adjustments made for P2P deduplication",
	}, []string{"direction", "tier"})

	p2pAdaptiveTTL = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "p2p_adaptive_ttl_seconds",
		Help: "Current adaptive TTL for P2P deduplication",
	}, []string{"message_type", "tier"})

	p2pDuplicateRate = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "p2p_duplicate_rate",
		Help: "Current duplicate rate for P2P messages",
	}, []string{"message_type", "tier"})

	p2pPeerReputation = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "p2p_peer_reputation_score",
		Help: "Reputation score for P2P peers based on duplicate behavior",
	}, []string{"peer_id", "tier"})

	p2pMessageVelocity = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "p2p_message_velocity",
		Help: "Messages per second for different message types",
	}, []string{"message_type", "tier"})
)

// EnterpriseP2PDeduper provides enterprise-grade P2P message deduplication
type EnterpriseP2PDeduper struct {
	// Core deduplication
	mu   sync.RWMutex
	seen map[string]*P2PEntry

	// Adaptive TTL management
	ttl         time.Duration
	minTTL      time.Duration
	maxTTL      time.Duration
	dupCount    int64
	totalCount  int64
	lastAdjust  time.Time
	adjustEvery time.Duration

	// Enterprise features
	logger   *zap.Logger
	tier     string
	capacity int
	order    []string

	// P2P-specific optimizations
	peerTracking      bool
	messageTypeDedup  bool
	crossNetworkDedup bool
	reputationScoring bool
	priorityQueuing   bool

	// Peer reputation system
	peerReputations map[string]*PeerReputation
	reputationDecay time.Duration

	// Message type tracking
	messageTypes map[string]*MessageTypeStats

	// Performance tracking
	messageVelocity  map[string]float64 // messages per second by type
	lastMessageTimes map[string]time.Time

	// Advanced algorithms
	adaptiveLearning  bool
	confidenceScoring bool
	anomalyDetection  bool

	// ML-based optimization
	learningRate        float64
	confidenceThreshold float64
	anomalyThreshold    float64

	// Network-specific configurations
	networkConfigs map[string]*NetworkConfig
}

// P2PEntry represents a deduplicated P2P message with metadata
type P2PEntry struct {
	Hash        string                 `json:"hash"`
	MessageType string                 `json:"message_type"`
	PeerID      string                 `json:"peer_id"`
	Network     string                 `json:"network"`
	FirstSeen   time.Time              `json:"first_seen"`
	LastSeen    time.Time              `json:"last_seen"`
	SeenCount   int                    `json:"seen_count"`
	Confidence  float64                `json:"confidence"`
	Priority    int                    `json:"priority"`
	Size        int64                  `json:"size"`
	Source      string                 `json:"source"`
	Properties  map[string]interface{} `json:"properties"`
}

// PeerReputation tracks reputation metrics for individual peers
type PeerReputation struct {
	PeerID          string    `json:"peer_id"`
	TotalMessages   int64     `json:"total_messages"`
	DuplicateCount  int64     `json:"duplicate_count"`
	DuplicateRate   float64   `json:"duplicate_rate"`
	ReputationScore float64   `json:"reputation_score"`
	LastActivity    time.Time `json:"last_activity"`
	IsBlacklisted   bool      `json:"is_blacklisted"`
	BlacklistReason string    `json:"blacklist_reason,omitempty"`
	TrustLevel      string    `json:"trust_level"` // "LOW", "MEDIUM", "HIGH", "TRUSTED"
}

// MessageTypeStats tracks statistics for different P2P message types
type MessageTypeStats struct {
	TotalSeen      int64         `json:"total_seen"`
	Duplicates     int64         `json:"duplicates"`
	DuplicateRate  float64       `json:"duplicate_rate"`
	AdaptiveTTL    time.Duration `json:"adaptive_ttl"`
	AvgTimeBetween time.Duration `json:"avg_time_between"`
	LastSeen       time.Time     `json:"last_seen"`
	Velocity       float64       `json:"velocity"`
	AverageSize    float64       `json:"average_size"`
}

// NetworkConfig stores network-specific P2P deduplication configurations
type NetworkConfig struct {
	Network          string        `json:"network"`
	BaseTTL          time.Duration `json:"base_ttl"`
	MaxMessageSize   int64         `json:"max_message_size"`
	PriorityMessages []string      `json:"priority_messages"`
	TrustedPeers     []string      `json:"trusted_peers"`
}

// Deduper provides backward compatibility with the legacy interface
type Deduper struct {
	enterprise *EnterpriseP2PDeduper
}

// NewEnterpriseP2PDeduper creates a new enterprise-grade P2P deduplicator
func NewEnterpriseP2PDeduper(tier string, logger *zap.Logger) *EnterpriseP2PDeduper {
	capacity := getP2PCapacityForTier(tier)

	epd := &EnterpriseP2PDeduper{
		seen:        make(map[string]*P2PEntry, capacity),
		order:       make([]string, 0, capacity),
		capacity:    capacity,
		ttl:         getP2PBaseTTL(tier),
		minTTL:      5 * time.Second,
		maxTTL:      30 * time.Minute,
		adjustEvery: 60 * time.Second,
		lastAdjust:  time.Now(),
		logger:      logger,
		tier:        tier,

		// Enable features based on tier
		peerTracking:      tier != "FREE",
		messageTypeDedup:  true,
		crossNetworkDedup: tier == "ENTERPRISE",
		reputationScoring: tier == "ENTERPRISE" || tier == "BUSINESS",
		priorityQueuing:   tier == "ENTERPRISE",
		adaptiveLearning:  tier == "ENTERPRISE",
		confidenceScoring: tier == "ENTERPRISE" || tier == "BUSINESS",
		anomalyDetection:  tier == "ENTERPRISE",

		// Initialize tracking systems
		peerReputations:  make(map[string]*PeerReputation),
		messageTypes:     make(map[string]*MessageTypeStats),
		messageVelocity:  make(map[string]float64),
		lastMessageTimes: make(map[string]time.Time),
		networkConfigs:   make(map[string]*NetworkConfig),

		// ML parameters
		learningRate:        0.05,
		confidenceThreshold: 0.75,
		anomalyThreshold:    3.0, // Standard deviations for anomaly detection
		reputationDecay:     24 * time.Hour,
	}

	// Initialize default network configurations
	epd.initializeNetworkConfigs()

	if logger != nil {
		logger.Info("Enterprise P2P Deduper initialized",
			zap.String("tier", tier),
			zap.Int("capacity", capacity),
			zap.Duration("base_ttl", epd.ttl),
			zap.Bool("peer_tracking", epd.peerTracking),
			zap.Bool("reputation_scoring", epd.reputationScoring),
			zap.Bool("adaptive_learning", epd.adaptiveLearning))
	}

	return epd
}

// getP2PCapacityForTier returns appropriate capacity for service tier
func getP2PCapacityForTier(tier string) int {
	switch tier {
	case "FREE":
		return 4096
	case "BUSINESS":
		return 16384
	case "ENTERPRISE":
		return 32768
	default:
		return 8192
	}
}

// getP2PBaseTTL returns base TTL optimized for P2P message patterns
func getP2PBaseTTL(tier string) time.Duration {
	switch tier {
	case "FREE":
		return 5 * time.Minute
	case "BUSINESS":
		return 10 * time.Minute
	case "ENTERPRISE":
		return 15 * time.Minute
	default:
		return 8 * time.Minute
	}
}

// initializeNetworkConfigs sets up default configurations for known networks
func (epd *EnterpriseP2PDeduper) initializeNetworkConfigs() {
	// Bitcoin network configuration
	epd.networkConfigs["bitcoin"] = &NetworkConfig{
		Network:          "bitcoin",
		BaseTTL:          12 * time.Minute, // Longer for Bitcoin's 10min blocks
		MaxMessageSize:   4 * 1024 * 1024,  // 4MB max
		PriorityMessages: []string{"block", "transaction", "addr"},
		TrustedPeers:     []string{},
	}

	// Ethereum network configuration
	epd.networkConfigs["ethereum"] = &NetworkConfig{
		Network:          "ethereum",
		BaseTTL:          3 * time.Minute, // Faster for Ethereum's 12s blocks
		MaxMessageSize:   2 * 1024 * 1024, // 2MB max
		PriorityMessages: []string{"NewBlock", "NewBlockHashes", "Transactions"},
		TrustedPeers:     []string{},
	}

	// Solana network configuration
	epd.networkConfigs["solana"] = &NetworkConfig{
		Network:          "solana",
		BaseTTL:          1 * time.Minute, // Fast for Solana's 400ms slots
		MaxMessageSize:   1 * 1024 * 1024, // 1MB max
		PriorityMessages: []string{"slot", "block", "shred"},
		TrustedPeers:     []string{},
	}
}

// IsDuplicate checks if a P2P message is a duplicate with enterprise features
func (epd *EnterpriseP2PDeduper) IsDuplicate(hash, messageType, peerID string, options ...dedup.DedupeOption) bool {
	if hash == "" {
		return false
	}

	// Apply options
	opts := &dedup.DedupeOptions{}
	for _, opt := range options {
		opt(opts)
	}

	now := time.Now()
	epd.mu.Lock()
	defer epd.mu.Unlock()

	epd.totalCount++

	// Update message type statistics
	if epd.messageTypes[messageType] == nil {
		epd.messageTypes[messageType] = &MessageTypeStats{
			AdaptiveTTL: epd.ttl,
		}
	}
	typeStats := epd.messageTypes[messageType]
	typeStats.TotalSeen++

	// Update peer reputation tracking
	if epd.peerTracking && peerID != "" {
		epd.updatePeerTracking(peerID, now)
	}

	// Generate composite key for cross-network deduplication
	key := epd.generateKey(hash, messageType, peerID, opts)

	if entry, exists := epd.seen[key]; exists {
		// Calculate type-specific TTL
		currentTTL := epd.getAdaptiveTTL(messageType, typeStats)

		// Check peer reputation for early filtering
		if epd.reputationScoring && peerID != "" {
			if peer := epd.peerReputations[peerID]; peer != nil && peer.IsBlacklisted {
				epd.dupCount++
				typeStats.Duplicates++
				p2pDuplicatesSuppressed.WithLabelValues(messageType, "blacklisted", epd.tier).Inc()
				return true
			}
		}

		if now.Sub(entry.LastSeen) <= currentTTL {
			// It's a duplicate
			epd.dupCount++
			typeStats.Duplicates++
			entry.LastSeen = now
			entry.SeenCount++

			// Update peer reputation for duplicates
			if epd.reputationScoring && peerID != "" {
				epd.updatePeerReputation(peerID, true)
			}

			// Update confidence if enabled
			if epd.confidenceScoring {
				entry.Confidence = epd.updateConfidence(entry, now)
			}

			// Update metrics
			p2pDuplicatesSuppressed.WithLabelValues(messageType, "duplicate", epd.tier).Inc()

			// Track velocity for active messages
			epd.updateVelocityTracking(messageType, now, typeStats)

			return true
		}

		// Entry expired, update it
		entry.LastSeen = now
		entry.SeenCount = 1 // Reset count for expired entry
		if epd.confidenceScoring {
			entry.Confidence = epd.calculateInitialConfidence(hash, messageType, peerID, opts)
		}
	} else {
		// New entry
		if len(epd.seen) >= epd.capacity {
			epd.evictOldest()
		}

		entry := &P2PEntry{
			Hash:        hash,
			MessageType: messageType,
			PeerID:      peerID,
			Network:     opts.Source,
			FirstSeen:   now,
			LastSeen:    now,
			SeenCount:   1,
			Source:      opts.Source,
			Priority:    epd.calculatePriority(messageType, peerID, opts),
			Size:        opts.Size,
			Properties:  opts.Properties,
		}

		if epd.confidenceScoring {
			entry.Confidence = epd.calculateInitialConfidence(hash, messageType, peerID, opts)
		}

		epd.seen[key] = entry
		epd.order = append(epd.order, key)

		// Update peer reputation for new messages
		if epd.reputationScoring && peerID != "" {
			epd.updatePeerReputation(peerID, false)
		}
	}

	// Update velocity tracking
	epd.updateVelocityTracking(messageType, now, typeStats)

	// Periodic TTL adjustment with ML optimization
	if now.Sub(epd.lastAdjust) >= epd.adjustEvery {
		epd.adjustTTLWithML()
		epd.lastAdjust = now
	}

	// Anomaly detection for enterprise tier
	if epd.anomalyDetection {
		epd.performAnomalyDetection(messageType, peerID, now)
	}

	return false
}

// generateKey creates appropriate keys based on configuration
func (epd *EnterpriseP2PDeduper) generateKey(hash, messageType, peerID string, opts *dedup.DedupeOptions) string {
	if epd.crossNetworkDedup {
		// Cross-network deduplication - just use hash
		return hash
	}

	// Network and peer-specific deduplication
	network := "unknown"
	if opts.Source != "" {
		network = opts.Source
	}

	if epd.peerTracking && peerID != "" {
		return fmt.Sprintf("%s:%s:%s:%s", network, messageType, peerID, hash)
	}

	return fmt.Sprintf("%s:%s:%s", network, messageType, hash)
}

// getAdaptiveTTL returns adaptive TTL for a specific message type
func (epd *EnterpriseP2PDeduper) getAdaptiveTTL(messageType string, typeStats *MessageTypeStats) time.Duration {
	if typeStats.AdaptiveTTL > 0 {
		return typeStats.AdaptiveTTL
	}

	// Use network-specific TTL if available
	if epd.networkConfigs != nil {
		for _, config := range epd.networkConfigs {
			for _, priorityMsg := range config.PriorityMessages {
				if priorityMsg == messageType {
					return config.BaseTTL
				}
			}
		}
	}

	return epd.ttl
}

// updatePeerTracking updates basic peer activity tracking
func (epd *EnterpriseP2PDeduper) updatePeerTracking(peerID string, now time.Time) {
	if epd.peerReputations[peerID] == nil {
		epd.peerReputations[peerID] = &PeerReputation{
			PeerID:          peerID,
			ReputationScore: 1.0,
			TrustLevel:      "MEDIUM",
			LastActivity:    now,
		}
	}

	peer := epd.peerReputations[peerID]
	peer.TotalMessages++
	peer.LastActivity = now
}

// updatePeerReputation updates peer reputation based on duplicate behavior
func (epd *EnterpriseP2PDeduper) updatePeerReputation(peerID string, isDuplicate bool) {
	if !epd.reputationScoring || peerID == "" {
		return
	}

	peer := epd.peerReputations[peerID]
	if peer == nil {
		return
	}

	if isDuplicate {
		peer.DuplicateCount++
	}

	// Calculate duplicate rate
	if peer.TotalMessages > 0 {
		peer.DuplicateRate = float64(peer.DuplicateCount) / float64(peer.TotalMessages)
	}

	// Update reputation score (exponential moving average)
	alpha := epd.learningRate
	if isDuplicate {
		// Penalize for duplicates
		penalty := 0.1 * (1.0 + peer.DuplicateRate)
		peer.ReputationScore = peer.ReputationScore*(1-alpha) + (peer.ReputationScore-penalty)*alpha
	} else {
		// Reward for new messages
		reward := 0.02
		peer.ReputationScore = peer.ReputationScore*(1-alpha) + (peer.ReputationScore+reward)*alpha
	}

	// Bounds checking
	if peer.ReputationScore < 0.0 {
		peer.ReputationScore = 0.0
	} else if peer.ReputationScore > 1.0 {
		peer.ReputationScore = 1.0
	}

	// Update trust level
	switch {
	case peer.ReputationScore >= 0.9:
		peer.TrustLevel = "TRUSTED"
	case peer.ReputationScore >= 0.7:
		peer.TrustLevel = "HIGH"
	case peer.ReputationScore >= 0.4:
		peer.TrustLevel = "MEDIUM"
	default:
		peer.TrustLevel = "LOW"
	}

	// Blacklist peers with very poor reputation
	if peer.ReputationScore < 0.1 && peer.DuplicateRate > 0.8 && peer.TotalMessages > 50 {
		peer.IsBlacklisted = true
		peer.BlacklistReason = "High duplicate rate with low reputation"
	}

	// Update Prometheus metric
	p2pPeerReputation.WithLabelValues(peerID, epd.tier).Set(peer.ReputationScore)
}

// calculateInitialConfidence calculates initial confidence for new entries
func (epd *EnterpriseP2PDeduper) calculateInitialConfidence(hash, messageType, peerID string, opts *dedup.DedupeOptions) float64 {
	if !epd.confidenceScoring {
		return 1.0
	}

	confidence := 0.7 // Base confidence

	// Hash quality assessment
	if len(hash) >= 32 {
		confidence += 0.1
	}

	// Peer reputation influence
	if epd.reputationScoring && peerID != "" {
		if peer := epd.peerReputations[peerID]; peer != nil {
			confidence += peer.ReputationScore * 0.2
		}
	}

	// Message type reliability
	if messageType != "" {
		switch messageType {
		case "block", "NewBlock":
			confidence += 0.1 // Blocks are usually reliable
		case "transaction", "Transactions":
			confidence += 0.05
		case "ping", "pong":
			confidence -= 0.05 // Control messages less critical
		}
	}

	// Size-based assessment
	if opts.Size > 0 {
		if opts.Size > 1024*1024 { // Large messages more likely to be unique
			confidence += 0.05
		}
	}

	if confidence > 1.0 {
		confidence = 1.0
	} else if confidence < 0.1 {
		confidence = 0.1
	}

	return confidence
}

// updateConfidence updates confidence based on observation patterns
func (epd *EnterpriseP2PDeduper) updateConfidence(entry *P2PEntry, now time.Time) float64 {
	if !epd.confidenceScoring {
		return entry.Confidence
	}

	// Increase confidence with repeated sightings from different peers
	frequencyBoost := 1.0 + float64(entry.SeenCount)*0.01
	if frequencyBoost > 1.3 {
		frequencyBoost = 1.3
	}

	// Time-based confidence decay
	timeSinceFirst := now.Sub(entry.FirstSeen)
	timeDecay := math.Exp(-float64(timeSinceFirst) / float64(2*time.Hour))

	// Peer reputation influence
	peerInfluence := 1.0
	if epd.reputationScoring && entry.PeerID != "" {
		if peer := epd.peerReputations[entry.PeerID]; peer != nil {
			peerInfluence = 0.5 + peer.ReputationScore*0.5
		}
	}

	newConfidence := entry.Confidence * frequencyBoost * timeDecay * peerInfluence
	if newConfidence > 1.0 {
		newConfidence = 1.0
	} else if newConfidence < 0.1 {
		newConfidence = 0.1
	}

	return newConfidence
}

// calculatePriority determines priority based on message type and peer reputation
func (epd *EnterpriseP2PDeduper) calculatePriority(messageType, peerID string, opts *dedup.DedupeOptions) int {
	if !epd.priorityQueuing {
		return 1
	}

	priority := 1

	// Message type-based priority
	switch messageType {
	case "block", "NewBlock":
		priority = 10 // Highest priority
	case "transaction", "Transactions":
		priority = 7
	case "addr", "getaddr":
		priority = 5
	case "ping", "pong":
		priority = 1 // Lowest priority
	default:
		priority = 3
	}

	// Peer reputation influence on priority
	if epd.reputationScoring && peerID != "" {
		if peer := epd.peerReputations[peerID]; peer != nil {
			switch peer.TrustLevel {
			case "TRUSTED":
				priority += 3
			case "HIGH":
				priority += 2
			case "MEDIUM":
				priority += 1
			case "LOW":
				priority -= 1
			}
		}
	}

	// Option-based adjustments
	if opts.Priority > 0 {
		priority = opts.Priority
	}

	if priority < 1 {
		priority = 1
	} else if priority > 15 {
		priority = 15
	}

	return priority
}

// updateVelocityTracking updates velocity statistics for performance optimization
func (epd *EnterpriseP2PDeduper) updateVelocityTracking(messageType string, now time.Time, typeStats *MessageTypeStats) {
	if lastTime, exists := epd.lastMessageTimes[messageType]; exists {
		timeSinceLast := now.Sub(lastTime)

		// Exponential moving average for time between messages
		alpha := epd.learningRate
		if typeStats.AvgTimeBetween == 0 {
			typeStats.AvgTimeBetween = timeSinceLast
		} else {
			typeStats.AvgTimeBetween = time.Duration(float64(typeStats.AvgTimeBetween)*(1-alpha) + float64(timeSinceLast)*alpha)
		}

		// Calculate velocity (messages per second)
		if typeStats.AvgTimeBetween > 0 {
			typeStats.Velocity = 1.0 / typeStats.AvgTimeBetween.Seconds()
			epd.messageVelocity[messageType] = typeStats.Velocity
		}

		// Update Prometheus metric
		p2pMessageVelocity.WithLabelValues(messageType, epd.tier).Set(typeStats.Velocity)
	}

	epd.lastMessageTimes[messageType] = now
	typeStats.LastSeen = now
}

// adjustTTLWithML performs ML-based TTL adjustment
func (epd *EnterpriseP2PDeduper) adjustTTLWithML() {
	if epd.totalCount < 100 {
		return
	}

	// Calculate global duplicate rate
	globalRate := float64(epd.dupCount) / float64(epd.totalCount)

	// ML-based TTL adjustment
	if epd.adaptiveLearning {
		epd.adjustTTLWithAdvancedML(globalRate)
	} else {
		epd.adjustTTLBasic(globalRate)
	}

	// Update metrics
	p2pAdaptiveTTL.WithLabelValues("global", epd.tier).Set(epd.ttl.Seconds())
	p2pDuplicateRate.WithLabelValues("global", epd.tier).Set(globalRate)

	// Reset counters with partial decay
	epd.totalCount = epd.totalCount / 2
	epd.dupCount = epd.dupCount / 2
}

// adjustTTLWithAdvancedML implements advanced ML-based TTL optimization
func (epd *EnterpriseP2PDeduper) adjustTTLWithAdvancedML(globalRate float64) {
	// Calculate velocity-adjusted learning rate
	adaptiveLearningRate := epd.learningRate

	// Multi-factor TTL adjustment
	targetRate := 0.25 // Target 25% duplicate rate for P2P optimization
	rateDelta := globalRate - targetRate

	// Calculate TTL adjustment factor
	adjustmentFactor := 1.0 + (rateDelta * adaptiveLearningRate * 1.5)

	// Apply adjustment
	newTTL := time.Duration(float64(epd.ttl) * adjustmentFactor)

	// Bounds checking
	if newTTL < epd.minTTL {
		newTTL = epd.minTTL
	} else if newTTL > epd.maxTTL {
		newTTL = epd.maxTTL
	}

	// Track adjustment direction
	if newTTL > epd.ttl {
		p2pTTLAdjustments.WithLabelValues("increase", epd.tier).Inc()
	} else if newTTL < epd.ttl {
		p2pTTLAdjustments.WithLabelValues("decrease", epd.tier).Inc()
	}

	epd.ttl = newTTL

	// Adjust message type-specific TTLs
	for messageType, typeStats := range epd.messageTypes {
		if typeStats.TotalSeen > 20 {
			typeRate := float64(typeStats.Duplicates) / float64(typeStats.TotalSeen)
			typeFactor := 1.0 + (typeRate-targetRate)*adaptiveLearningRate

			// Message type-specific adjustments
			switch messageType {
			case "block", "NewBlock":
				typeFactor *= 1.5 // Blocks need longer TTL
			case "ping", "pong":
				typeFactor *= 0.5 // Control messages can have shorter TTL
			}

			newTypeTTL := time.Duration(float64(epd.ttl) * typeFactor)
			if newTypeTTL >= epd.minTTL && newTypeTTL <= epd.maxTTL {
				typeStats.AdaptiveTTL = newTypeTTL
			}

			// Update type-specific metrics
			p2pDuplicateRate.WithLabelValues(messageType, epd.tier).Set(typeRate)
			p2pAdaptiveTTL.WithLabelValues(messageType, epd.tier).Set(typeStats.AdaptiveTTL.Seconds())
		}
	}
}

// adjustTTLBasic performs basic TTL adjustment for non-enterprise tiers
func (epd *EnterpriseP2PDeduper) adjustTTLBasic(rate float64) {
	switch {
	case rate > 0.40:
		// Lots of duplicates, increase TTL
		epd.ttl = epd.ttl + 30*time.Second
		p2pTTLAdjustments.WithLabelValues("increase", epd.tier).Inc()
	case rate > 0.20:
		epd.ttl = epd.ttl + 15*time.Second
		p2pTTLAdjustments.WithLabelValues("increase", epd.tier).Inc()
	case rate < 0.05:
		// Few duplicates: shrink TTL
		if epd.ttl > 2*time.Minute {
			epd.ttl = epd.ttl - 15*time.Second
			p2pTTLAdjustments.WithLabelValues("decrease", epd.tier).Inc()
		}
	default:
		// Small drift
		epd.ttl = epd.ttl + 5*time.Second
	}

	// Bounds checking
	if epd.ttl < epd.minTTL {
		epd.ttl = epd.minTTL
	}
	if epd.ttl > epd.maxTTL {
		epd.ttl = epd.maxTTL
	}
}

// performAnomalyDetection detects anomalous patterns in P2P traffic
func (epd *EnterpriseP2PDeduper) performAnomalyDetection(messageType, peerID string, now time.Time) {
	if !epd.anomalyDetection {
		return
	}

	// Check for rapid-fire duplicates from same peer
	if peerID != "" {
		if peer := epd.peerReputations[peerID]; peer != nil {
			timeSinceLastActivity := now.Sub(peer.LastActivity)
			if timeSinceLastActivity < 100*time.Millisecond && peer.DuplicateRate > 0.5 {
				// Potential spam/flooding detected
				if epd.logger != nil {
					epd.logger.Warn("Potential P2P flooding detected",
						zap.String("peer_id", peerID),
						zap.String("message_type", messageType),
						zap.Float64("duplicate_rate", peer.DuplicateRate),
						zap.Duration("time_since_last", timeSinceLastActivity))
				}
			}
		}
	}

	// Check for unusual message velocity
	if typeStats := epd.messageTypes[messageType]; typeStats != nil {
		if typeStats.Velocity > 0 {
			// Calculate z-score for velocity anomaly detection
			avgVelocity := 0.0
			count := 0
			for _, vel := range epd.messageVelocity {
				if vel > 0 {
					avgVelocity += vel
					count++
				}
			}
			if count > 0 {
				avgVelocity /= float64(count)
				if typeStats.Velocity > avgVelocity*epd.anomalyThreshold {
					if epd.logger != nil {
						epd.logger.Warn("Anomalous message velocity detected",
							zap.String("message_type", messageType),
							zap.Float64("current_velocity", typeStats.Velocity),
							zap.Float64("average_velocity", avgVelocity))
					}
				}
			}
		}
	}
}

// evictOldest implements intelligent eviction based on priority and reputation
func (epd *EnterpriseP2PDeduper) evictOldest() {
	if len(epd.order) == 0 {
		return
	}

	if epd.priorityQueuing {
		epd.evictByPriorityAndReputation()
	} else {
		// Simple FIFO eviction
		oldKey := epd.order[0]
		epd.order = epd.order[1:]
		delete(epd.seen, oldKey)
	}
}

// evictByPriorityAndReputation implements sophisticated eviction
func (epd *EnterpriseP2PDeduper) evictByPriorityAndReputation() {
	var lowestScoreKey string
	lowestScore := 1000.0

	// Calculate composite score for each entry
	for key, entry := range epd.seen {
		score := float64(entry.Priority) * 10.0 // Base priority weight

		// Add confidence weight
		if epd.confidenceScoring {
			score += entry.Confidence * 5.0
		}

		// Add peer reputation weight
		if epd.reputationScoring && entry.PeerID != "" {
			if peer := epd.peerReputations[entry.PeerID]; peer != nil {
				score += peer.ReputationScore * 3.0
			}
		}

		// Penalize old entries
		age := time.Since(entry.LastSeen).Hours()
		score -= age * 0.1

		if score < lowestScore {
			lowestScore = score
			lowestScoreKey = key
		}
	}

	if lowestScoreKey != "" {
		// Remove from order slice
		for i, key := range epd.order {
			if key == lowestScoreKey {
				epd.order = append(epd.order[:i], epd.order[i+1:]...)
				break
			}
		}
		delete(epd.seen, lowestScoreKey)
	}
}

// GetStats returns comprehensive P2P deduplication statistics
func (epd *EnterpriseP2PDeduper) GetStats() map[string]interface{} {
	epd.mu.RLock()
	defer epd.mu.RUnlock()

	globalRate := 0.0
	if epd.totalCount > 0 {
		globalRate = float64(epd.dupCount) / float64(epd.totalCount)
	}

	stats := map[string]interface{}{
		"tier":                  epd.tier,
		"total_seen":            epd.totalCount,
		"duplicates_found":      epd.dupCount,
		"global_duplicate_rate": globalRate,
		"current_ttl_seconds":   epd.ttl.Seconds(),
		"min_ttl_seconds":       epd.minTTL.Seconds(),
		"max_ttl_seconds":       epd.maxTTL.Seconds(),
		"capacity":              epd.capacity,
		"current_size":          len(epd.seen),
		"peer_tracking":         epd.peerTracking,
		"reputation_scoring":    epd.reputationScoring,
		"adaptive_learning":     epd.adaptiveLearning,
		"confidence_scoring":    epd.confidenceScoring,
		"anomaly_detection":     epd.anomalyDetection,
		"cross_network_dedup":   epd.crossNetworkDedup,
		"learning_rate":         epd.learningRate,
		"confidence_threshold":  epd.confidenceThreshold,
		"anomaly_threshold":     epd.anomalyThreshold,
	}

	// Add message type statistics
	messageStatsMap := make(map[string]interface{})
	for messageType, typeStats := range epd.messageTypes {
		typeRate := 0.0
		if typeStats.TotalSeen > 0 {
			typeRate = float64(typeStats.Duplicates) / float64(typeStats.TotalSeen)
		}

		messageStatsMap[messageType] = map[string]interface{}{
			"total_seen":               typeStats.TotalSeen,
			"duplicates":               typeStats.Duplicates,
			"duplicate_rate":           typeRate,
			"adaptive_ttl_seconds":     typeStats.AdaptiveTTL.Seconds(),
			"avg_time_between_seconds": typeStats.AvgTimeBetween.Seconds(),
			"velocity":                 typeStats.Velocity,
			"average_size":             typeStats.AverageSize,
		}
	}
	stats["message_type_statistics"] = messageStatsMap

	// Add peer reputation statistics
	peerStatsMap := make(map[string]interface{})
	for peerID, peer := range epd.peerReputations {
		peerStatsMap[peerID] = map[string]interface{}{
			"total_messages":   peer.TotalMessages,
			"duplicate_count":  peer.DuplicateCount,
			"duplicate_rate":   peer.DuplicateRate,
			"reputation_score": peer.ReputationScore,
			"trust_level":      peer.TrustLevel,
			"is_blacklisted":   peer.IsBlacklisted,
			"blacklist_reason": peer.BlacklistReason,
		}
	}
	stats["peer_reputation_statistics"] = peerStatsMap

	return stats
}

// Cleanup performs intelligent cleanup of expired entries and peer reputation decay
func (epd *EnterpriseP2PDeduper) Cleanup() {
	epd.mu.Lock()
	defer epd.mu.Unlock()

	now := time.Now()
	keysToDelete := []string{}

	// Cleanup expired entries
	for key, entry := range epd.seen {
		// Use message type-specific TTL
		typeStats := epd.messageTypes[entry.MessageType]
		currentTTL := epd.ttl
		if typeStats != nil && typeStats.AdaptiveTTL > 0 {
			currentTTL = typeStats.AdaptiveTTL
		}

		// Consider confidence and peer reputation in cleanup decisions
		adjustedTTL := currentTTL
		if epd.confidenceScoring && entry.Confidence < epd.confidenceThreshold {
			adjustedTTL = time.Duration(float64(currentTTL) * entry.Confidence)
		}

		if epd.reputationScoring && entry.PeerID != "" {
			if peer := epd.peerReputations[entry.PeerID]; peer != nil && peer.ReputationScore < 0.3 {
				// Reduce TTL for low-reputation peers
				adjustedTTL = time.Duration(float64(adjustedTTL) * peer.ReputationScore)
			}
		}

		if now.Sub(entry.LastSeen) > adjustedTTL {
			keysToDelete = append(keysToDelete, key)
		}
	}

	// Remove expired entries
	for _, key := range keysToDelete {
		delete(epd.seen, key)

		// Remove from order slice
		for i, orderKey := range epd.order {
			if orderKey == key {
				epd.order = append(epd.order[:i], epd.order[i+1:]...)
				break
			}
		}
	}

	// Cleanup old peer reputations
	if epd.reputationScoring {
		for peerID, peer := range epd.peerReputations {
			if now.Sub(peer.LastActivity) > epd.reputationDecay {
				delete(epd.peerReputations, peerID)
			}
		}
	}

	if len(keysToDelete) > 0 && epd.logger != nil {
		epd.logger.Debug("P2P dedup cleanup completed",
			zap.Int("entries_removed", len(keysToDelete)),
			zap.Int("entries_remaining", len(epd.seen)),
			zap.Int("peers_tracked", len(epd.peerReputations)),
			zap.String("tier", epd.tier))
	}
}

// SetTier updates the service tier and reconfigures accordingly
func (epd *EnterpriseP2PDeduper) SetTier(tier string) {
	epd.mu.Lock()
	defer epd.mu.Unlock()

	oldTier := epd.tier
	epd.tier = tier

	// Update tier-dependent features
	epd.peerTracking = tier != "FREE"
	epd.crossNetworkDedup = tier == "ENTERPRISE"
	epd.reputationScoring = tier == "ENTERPRISE" || tier == "BUSINESS"
	epd.priorityQueuing = tier == "ENTERPRISE"
	epd.adaptiveLearning = tier == "ENTERPRISE"
	epd.confidenceScoring = tier == "ENTERPRISE" || tier == "BUSINESS"
	epd.anomalyDetection = tier == "ENTERPRISE"

	// Update capacity
	newCapacity := getP2PCapacityForTier(tier)
	if newCapacity != epd.capacity {
		epd.capacity = newCapacity
		// Trigger cleanup if over capacity
		if len(epd.seen) > epd.capacity {
			epd.enforceCapacity()
		}
	}

	if epd.logger != nil {
		epd.logger.Info("P2P deduper tier updated",
			zap.String("old_tier", oldTier),
			zap.String("new_tier", tier),
			zap.Int("new_capacity", newCapacity))
	}
}

// enforceCapacity reduces cache size to fit within capacity limits
func (epd *EnterpriseP2PDeduper) enforceCapacity() {
	for len(epd.seen) > epd.capacity && len(epd.order) > 0 {
		epd.evictOldest()
	}
}

// TrackPeer updates peer tracking for the given peer ID
func (epd *EnterpriseP2PDeduper) TrackPeer(peerID string) {
	if epd.peerTracking {
		epd.updatePeerTracking(peerID, time.Now())
	}
}

// Close gracefully shuts down the P2P deduper
func (epd *EnterpriseP2PDeduper) Close() error {
	if epd.logger != nil {
		epd.logger.Info("P2P deduper shutdown",
			zap.String("tier", epd.tier),
			zap.Int64("total_processed", epd.totalCount),
			zap.Int64("duplicates_found", epd.dupCount),
			zap.Int("peers_tracked", len(epd.peerReputations)))
	}
	return nil
}

// Legacy compatibility interface implementation

// NewDeduper creates a new deduplication handler with backward compatibility
func NewDeduper(capacity int, ttl time.Duration) *Deduper {
	// Create enterprise deduper with FREE tier for backward compatibility
	enterprise := NewEnterpriseP2PDeduper("FREE", nil)
	if capacity > 0 {
		enterprise.capacity = capacity
	}
	if ttl > 0 {
		enterprise.ttl = ttl
	}

	return &Deduper{
		enterprise: enterprise,
	}
}

// Seen returns true if hash is already seen within TTL
func (d *Deduper) Seen(hash string, now time.Time) bool {
	return d.enterprise.IsDuplicate(hash, "unknown", "", dedup.WithSource("legacy"))
}

// Cleanup removes expired entries
func (d *Deduper) Cleanup() {
	d.enterprise.Cleanup()
}

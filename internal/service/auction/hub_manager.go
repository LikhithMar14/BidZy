package auction

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	types "github.com/LikhithMar14/BidZy/pkg/types"
)

type HubManager struct {
	hubs    map[string]*Hub
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

type HubStats struct {
	AuctionID   string `json:"auction_id"`
	ClientCount int    `json:"client_count"`
	HighestBid  *Bid   `json:"highest_bid,omitempty"`
}

type Stats struct {
	TotalHubs int        `json:"total_hubs"`
	Hubs      []HubStats `json:"hubs"`
	Timestamp time.Time  `json:"timestamp"`
}

type AllAuctionData struct {
	TotalAuctions int                  `json:"total_auctions"`
	Auctions      []*types.AuctionData `json:"auctions"`
	Timestamp     time.Time            `json:"timestamp"`
}

func NewHubManager() *HubManager {
	ctx, cancel := context.WithCancel(context.Background())
	manager := &HubManager{
		hubs:   make(map[string]*Hub),
		ctx:    ctx,
		cancel: cancel,
	}

	go manager.cleanupLoop()

	return manager
}

func (m *HubManager) GetHub(auctionId string) *Hub {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hubs[auctionId]
}

func (m *HubManager) GetOrCreateHub(auctionId string, increment float64, title string, description string, startingPrice int, duration time.Duration) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		log.Printf("HubManager is stopped, cannot create hub for auction %s", auctionId)
		return nil
	}

	if hub, exists := m.hubs[auctionId]; exists {
		log.Printf("Hub already exists for auction %s", auctionId)
		return hub
	}

	log.Printf("Creating new hub for auction %s", auctionId)
	hub := NewHub(auctionId, increment, title, description, startingPrice, duration)
	m.hubs[auctionId] = hub
	go hub.Run()

	go func() {
		log.Printf("Creating auction %s in database...", auctionId)
		time.Sleep(time.Second * 3)
		log.Printf("Auction %s created in database", auctionId)
	}()

	return hub
}

func (m *HubManager) CreateHub(auctionId, title, description string, startingPrice, increment float64, duration time.Duration) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		log.Printf("HubManager is stopped, cannot create hub for auction %s", auctionId)
		return nil
	}

	if _, exists := m.hubs[auctionId]; exists {
		log.Printf("Hub already exists for auction %s", auctionId)
		return nil
	}

	log.Printf("Creating new hub for auction %s with custom config", auctionId)
	hub := NewHub(auctionId, increment, title, description, int(startingPrice), duration)

	hub.Title = title
	hub.Description = description
	hub.StartingPrice = startingPrice
	hub.EndTime = hub.StartTime.Add(duration)

	m.hubs[auctionId] = hub
	go hub.Run()

	log.Printf("Created auction %s: %s", auctionId, title)
	return hub
}

func (m *HubManager) DeleteHub(auctionId string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, exists := m.hubs[auctionId]; exists {
		hub.Cancel()
		delete(m.hubs, auctionId)
		log.Printf("Hub %s deleted", auctionId)
	}
}

func (m *HubManager) GetStats() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var hubStats []HubStats
	for auctionID, hub := range m.hubs {
		stats := HubStats{
			AuctionID:   auctionID,
			ClientCount: hub.GetClientCount(),
			HighestBid:  hub.HighestBid,
		}
		hubStats = append(hubStats, stats)
	}

	stats := Stats{
		TotalHubs: len(m.hubs),
		Hubs:      hubStats,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(stats)
	if err != nil {
		log.Printf("Failed to marshal hub stats: %v", err)
		return "{\"error\":\"failed to marshal stats\"}"
	}
	return string(data)
}

func (m *HubManager) GetAllAuctionData() *AllAuctionData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	auctions := make([]*types.AuctionData, 0, len(m.hubs))
	for _, hub := range m.hubs {
		auctionData := hub.GetAuctionData()
		auctions = append(auctions, auctionData)
	}

	return &AllAuctionData{
		TotalAuctions: len(auctions),
		Auctions:      auctions,
		Timestamp:     time.Now(),
	}
}

func (m *HubManager) cleanupLoop() {
	const inactivityThreshold = 1 * time.Minute
	const cleanupInterval = 30 * time.Second

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	log.Printf("Starting hub cleanup loop")

	for {
		select {
		case <-m.ctx.Done():
			log.Printf("Hub cleanup loop stopping due to context cancellation")
			return
		case <-ticker.C:
			m.performCleanup(inactivityThreshold)
		}
	}
}

func (m *HubManager) performCleanup(inactivityThreshold time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	toDelete := make([]string, 0)

	for auctionID, hub := range m.hubs {
		if hub.GetClientCount() == 0 {
			lastActive := hub.GetLastActive()
			if now.Sub(lastActive) > inactivityThreshold {
				log.Printf("Marking inactive hub %s for cleanup (last active %v ago)",
					auctionID, now.Sub(lastActive))
				toDelete = append(toDelete, auctionID)
			}
		}
	}

	for _, auctionID := range toDelete {
		if hub, exists := m.hubs[auctionID]; exists {
			hub.Cancel()
			// See, here we are calling `hub.Cancel()`. I know that inside `Hub` we are not going to touch `m.mu.Lock()`."
			delete(m.hubs, auctionID)
			log.Printf("Cleaned up inactive hub %s", auctionID)
		}
	}

	if len(toDelete) > 0 {
		log.Printf("Cleanup completed: removed %d inactive hubs", len(toDelete))
	}
}

func (m *HubManager) Stop() {
	m.mu.Lock()
	m.stopped = true
	m.mu.Unlock()

	m.cancel()

	// Never call external functions (including cancel() or Cancel()) while holding a
	// mutex unless you know exactly what they do and can guarantee they won’t try to acquire the same mutex or block forever.

	m.mu.Lock()
	for _, hub := range m.hubs {
		hub.Cancel()
	}
	m.hubs = make(map[string]*Hub)
	m.mu.Unlock()

	// We call hub.Cancel() inside the lock assuming it doesn't access m.mu.
	// If it does, it can cause a deadlock since m.mu is already held.
	// Refactor to cancel hubs outside the lock if future changes modify HubManager.

	log.Printf("HubManager stopped")
}

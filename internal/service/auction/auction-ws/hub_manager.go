package auction_ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	auction "github.com/LikhithMar14/BidZy/internal/service/auction"
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

	// 	//cleanup when auction ends
	go manager.cleanupLoop()

	return manager

}

func (m *HubManager) GetHub(auctionId string) *Hub {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hubs[auctionId]
}

func (m *HubManager) GetOrCreateHub(auctionId string, increment float64, title string, description string, startingPrice int, startDateTime time.Time, endDateTime time.Time, duration time.Duration, auctionService *auction.AuctionHTTP) *Hub {
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

	fmt.Println("==== Recovering from Server Crash in GetOrCreateHub ====")

	fmt.Println("Starting Price:", startingPrice)
	fmt.Println("Increment:", increment)
	fmt.Println("Title:", title)
	fmt.Println("Description:", description)
	fmt.Println("Start DateTime:", startDateTime)
	fmt.Println("End DateTime:", endDateTime)
	fmt.Println("Duration:", duration)
	fmt.Println("Auction Service:", auctionService)

	log.Printf("Creating new hub for auction %s", auctionId)
	var RecoveredBid *Bid
	if startingPrice > 0 {
		RecoveredBid = &Bid{
			Price: float64(startingPrice),
			SenderID: "Recovered",
		}
	}

	hub := NewHub(auctionId, increment, title, description, startingPrice, startDateTime, endDateTime, duration, auctionService)
	hub.HighestBid = RecoveredBid
	m.hubs[auctionId] = hub
	go hub.Run()


	return hub
}

func (m *HubManager) CreateHub(auctionId, title, description string, startingPrice, increment float64, startDateTime time.Time, endDateTime time.Time, duration time.Duration, auctionService *auction.AuctionHTTP) *Hub {
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
	hub := NewHub(auctionId, increment, title, description, int(startingPrice), startDateTime, endDateTime, duration, auctionService)

	hub.Title = title
	hub.Description = description
	hub.StartingPrice = startingPrice
	hub.EndTime = endDateTime
	hub.StartTime = startDateTime

	m.hubs[auctionId] = hub
	go hub.Run()

	fmt.Println("Hub:", hub)

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

func (m *HubManager) HandleAuctionEnd(auctionID string) {
	go func() {
		log.Printf("Auction %s ended. Scheduling hub deletion in 30 seconds...", auctionID)
		time.Sleep(30 * time.Second)

		m.mu.Lock()
		defer m.mu.Unlock()

		if hub, exists := m.hubs[auctionID]; exists {
			hub.Cancel()
			delete(m.hubs, auctionID)
			log.Printf("Hub %s deleted after auction end grace period", auctionID)
		}
	}()
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

	now := time.Now().UTC()
	toDelete := make([]string, 0)

	for auctionID, hub := range m.hubs {
		if now.Before(hub.StartTime) {
			continue
		}
		if hub.GetClientCount() == 0 && now.After(hub.EndTime) {
			lastActive := hub.GetLastActive()
			if lastActive.IsZero() || now.Sub(lastActive) > inactivityThreshold {
				log.Printf("Marking hub %s for cleanup", auctionID)
				toDelete = append(toDelete, auctionID)
			}
		}
	}

	for _, auctionID := range toDelete {
		if hub, exists := m.hubs[auctionID]; exists {
			hub.Cancel()
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

	m.mu.Lock()
	for _, hub := range m.hubs {
		hub.Cancel()
	}
	m.hubs = make(map[string]*Hub)
	m.mu.Unlock()

	log.Printf("HubManager stopped")
}

func (m *HubManager) UpdateHubID(tempID, realID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, ok := m.hubs[tempID]; ok {
		delete(m.hubs, tempID)
		m.hubs[realID] = hub
	}
}

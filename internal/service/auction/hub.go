package auction

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	types "github.com/LikhithMar14/BidZy/pkg/types"
)

const (
	ChannelTimeout      = 5 * time.Second
	BroadcastTimeout    = 2 * time.Second
	RegistrationTimeout = 3 * time.Second
)

type Bid struct {
	SenderID  string    `json:"senderId"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

type Hub struct {
	AuctionID     string
	Title         string
	Description   string
	StartingPrice float64
	Clients       map[*Client]bool
	Register      chan *Client
	Unregister    chan *Client
	Bid           chan *Bid
	HighestBid    *Bid
	BidHistory    []*Bid
	Ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	lastActive    time.Time
	Increment     float64
	IsActive      bool
	StartTime     time.Time
	EndTime       time.Time
}

func NewHub(auctionID string, increment float64, title string, description string, startingPrice int, startDateTime time.Time, endDateTime time.Time, duration time.Duration) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		AuctionID:     auctionID,
		Title:         title,
		Description:   description,
		StartingPrice: float64(startingPrice),
		Clients:       make(map[*Client]bool),
		Register:      make(chan *Client, 10),
		Unregister:    make(chan *Client, 10),
		Bid:           make(chan *Bid, 100),
		BidHistory:    make([]*Bid, 0),
		Ctx:           ctx,
		cancel:        cancel,
		lastActive:    time.Now(),
		Increment:     increment,
		IsActive:      true,
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(duration),
	}
}

func (h *Hub) Run() {
	defer func() {
		log.Printf("Hub %s shutting down", h.AuctionID)
		h.closeAllClients()
	}()

	for {
		select {
		case <-h.Ctx.Done():
			log.Printf("Hub %s context cancelled", h.AuctionID)
			return

		case client := <-h.Register:
			h.handleClientRegistration(client)

		case client := <-h.Unregister:
			h.handleClientUnregistration(client)

		case bid := <-h.Bid:
			h.handleBid(bid)
		}
	}
}

func (h *Hub) handleClientRegistration(client *Client) {
	h.mu.Lock()
	h.Clients[client] = true
	h.lastActive = time.Now()
	clientCount := len(h.Clients)
	h.mu.Unlock()

	log.Printf("Client %s joined auction %s (total clients: %d)", client.ID, h.AuctionID, clientCount)

	userJoinedMsg := types.NewUserJoinedMessage(h.AuctionID, client.ID)
	if data, err := json.Marshal(userJoinedMsg); err == nil {
		h.broadcastToOthers(data, client.ID)
	}

	h.sendAuctionDataToClient(client)

	if h.HighestBid != nil {
		currentBidMsg := types.NewBidUpdateMessage(h.AuctionID, h.HighestBid.SenderID, h.HighestBid.Price)
		if data, err := json.Marshal(currentBidMsg); err == nil {
			h.sendToClientWithTimeout(client, data, RegistrationTimeout)
		}
	}

}

func (h *Hub) handleClientUnregistration(client *Client) {
	h.mu.Lock()
	_, existed := h.Clients[client]
	h.removeClientUnsafe(client)
	clientCount := len(h.Clients)
	h.mu.Unlock()

	if existed {
		log.Printf("Client %s left auction %s (remaining clients: %d)", client.ID, h.AuctionID, clientCount)

		userLeftMsg := types.NewUserLeftMessage(h.AuctionID, client.ID)
		if data, err := json.Marshal(userLeftMsg); err == nil {
			h.broadcastToOthers(data, client.ID)
		}
	}
}

func (h *Hub) handleBid(bid *Bid) {
	log.Printf("Received bid: %+v", bid)

	if bid == nil {
		log.Printf("Received nil bid in auction %s", h.AuctionID)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.IsActive || time.Now().After(h.EndTime) {
		rejectionMsg := types.NewErrorMessage(h.AuctionID, "Auction has ended")
		if data, err := json.Marshal(rejectionMsg); err == nil {
			h.sendToBidderUnsafe(data, bid.SenderID)
		}
		return
	}

	if bid.Price <= 0 {
		rejectionMsg := types.NewErrorMessage(h.AuctionID, "Bid amount must be positive")
		if data, err := json.Marshal(rejectionMsg); err == nil {
			h.sendToBidderUnsafe(data, bid.SenderID)
		}
		return
	}

	const maxBidAmount = 1_000_000_000
	if bid.Price > maxBidAmount {
		rejectionMsg := types.NewErrorMessage(h.AuctionID, "Bid amount too large")
		if data, err := json.Marshal(rejectionMsg); err == nil {
			h.sendToBidderUnsafe(data, bid.SenderID)
		}
		return
	}

	h.lastActive = time.Now()
	bid.Timestamp = time.Now()

	var minRequired float64
	if h.HighestBid != nil {
		minRequired = h.HighestBid.Price + h.Increment
	} else {
		minRequired = h.StartingPrice
	}

	if h.HighestBid != nil && h.HighestBid.SenderID == bid.SenderID {
		rejectionMsg := types.NewErrorMessage(h.AuctionID, "Cannot outbid yourself")
		if data, err := json.Marshal(rejectionMsg); err == nil {
			h.sendToBidderUnsafe(data, bid.SenderID)
		}
		h.BidHistory = append(h.BidHistory, bid)
		return
	}

	if bid.Price >= minRequired {
		h.HighestBid = bid
		h.BidHistory = append(h.BidHistory, bid)

		bidUpdateMsg := types.NewBidUpdateMessage(h.AuctionID, bid.SenderID, bid.Price)
		if data, err := json.Marshal(bidUpdateMsg); err == nil {
			h.broadcastUnsafe(data)
		} else {
			log.Printf("Failed to marshal bid update message in auction %s: %v", h.AuctionID, err)
		}

		log.Printf("New highest bid in auction %s: $%.2f by %s", h.AuctionID, bid.Price, bid.SenderID)
		return
	}

	h.BidHistory = append(h.BidHistory, bid)

	rejectionMsg := types.NewErrorMessage(h.AuctionID, fmt.Sprintf("Bid too low. Minimum required: $%.2f", minRequired))
	if data, err := json.Marshal(rejectionMsg); err == nil {
		h.sendToBidderUnsafe(data, bid.SenderID)
	} else {
		log.Printf("Failed to marshal bid rejection message: %v", err)
	}

	currentPrice := h.StartingPrice
	if h.HighestBid != nil {
		currentPrice = h.HighestBid.Price
	}
	log.Printf("Bid rejected in auction %s: $%.2f by %s (current: $%.2f, required: $%.2f)",
		h.AuctionID, bid.Price, bid.SenderID, currentPrice, minRequired)
}

func (h *Hub) broadcastUnsafe(message []byte) {
	log.Printf("Broadcasting message to clients: %s", string(message))
	for client := range h.Clients {
		select {
		case client.Send <- message:
		default:
			log.Printf("Failed to broadcast to client %s: channel full", client.ID)
			h.removeClientUnsafe(client)
		}
	}
}

func (h *Hub) broadcastToOthers(message []byte, excludeClientID string) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.Clients))
	for client := range h.Clients {
		if client.ID != excludeClientID {
			clients = append(clients, client)
		}
	}
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.Send <- message:
		case <-time.After(BroadcastTimeout):
			log.Printf("Broadcast timeout for client %s", client.ID)
			h.removeClient(client)
		}
	}
}

func (h *Hub) sendToBidderUnsafe(message []byte, bidderID string) {
	for client := range h.Clients {
		if client.ID == bidderID {
			select {
			case client.Send <- message:
			default:
				log.Printf("Failed to send message to bidder %s: channel full", client.ID)
				h.removeClientUnsafe(client)
			}
			break
		}
	}
}

func (h *Hub) sendToClientWithTimeout(client *Client, message []byte, timeout time.Duration) {
	select {
	case client.Send <- message:
	case <-time.After(timeout):
		log.Printf("Send timeout for client %s", client.ID)
		h.removeClient(client)
	}
}

func (h *Hub) sendAuctionDataToClient(client *Client) {
	h.mu.RLock()
	auctionData := &types.AuctionData{
		AuctionID:     h.AuctionID,
		Title:         h.Title,
		Description:   h.Description,
		StartingPrice: h.StartingPrice,
		CurrentPrice:  0,
		ClientCount:   len(h.Clients),
		IsActive:      h.IsActive,
		StartTime:     h.StartTime,
		EndTime:       h.EndTime,
		Increment:     h.Increment,
	}

	if h.HighestBid != nil {
		auctionData.CurrentPrice = h.HighestBid.Price
		auctionData.HighestBidder = h.HighestBid.SenderID
	}
	h.mu.RUnlock()

	auctionDataMsg := types.NewAuctionDataMessage(h.AuctionID, auctionData)
	if data, err := json.Marshal(auctionDataMsg); err == nil {
		h.sendToClientWithTimeout(client, data, RegistrationTimeout)
	} else {
		log.Printf("Failed to marshal auction data: %v", err)
	}
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeClientUnsafe(client)
}

func (h *Hub) removeClientUnsafe(client *Client) {
	if _, ok := h.Clients[client]; ok {
		delete(h.Clients, client)
		close(client.Send)
		h.lastActive = time.Now()
	}
}

func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.Clients {
		close(client.Send)
		if err := client.Conn.Close(); err != nil {
			log.Printf("Error closing connection for client %s: %v", client.ID, err)
		}
	}
	h.Clients = make(map[*Client]bool)
}

func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.Clients)
}

func (h *Hub) GetLastActive() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastActive
}

func (h *Hub) GetAuctionData() *types.AuctionData {
	h.mu.RLock()
	defer h.mu.RUnlock()

	currentPrice := h.StartingPrice
	var highestBidder string

	if h.HighestBid != nil {
		currentPrice = h.HighestBid.Price
		highestBidder = h.HighestBid.SenderID
	}

	auctionData := &types.AuctionData{
		AuctionID:     h.AuctionID,
		Title:         h.Title,
		Description:   h.Description,
		StartingPrice: h.StartingPrice,
		CurrentPrice:  currentPrice,
		HighestBidder: highestBidder,
		ClientCount:   len(h.Clients),
		IsActive:      h.IsActive && time.Now().Before(h.EndTime),
		StartTime:     h.StartTime,
		EndTime:       h.EndTime,
		Increment:     h.Increment,
	}

	return auctionData
}
func (h *Hub) Cancel() {
	log.Printf("Cancelling hub %s", h.AuctionID)
	h.mu.Lock()
	h.IsActive = false
	h.mu.Unlock()
	h.cancel()
}

type ClientInfo struct {
	ID       string    `json:"id"`
	JoinedAt time.Time `json:"joinedAt"`
	IsActive bool      `json:"isActive"`
}

func (h *Hub) GetClientInfo() []ClientInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clientList := make([]ClientInfo, 0, len(h.Clients))
	for client := range h.Clients {
		clientList = append(clientList, ClientInfo{
			ID:       client.ID,
			JoinedAt: time.Now(),
			IsActive: true,
		})
	}
	return clientList
}

func (h *Hub) GetBidHistory() []*Bid {
	h.mu.RLock()
	defer h.mu.RUnlock()

	history := make([]*Bid, len(h.BidHistory))
	copy(history, h.BidHistory)
	return history
}

package auction_ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/LikhithMar14/BidZy/internal/service/auction"
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
	UserName  *string   `json:"userName"`
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
	auctionHTTP   auction.AuctionHTTP
}

func NewHub(auctionID string, increment float64, title string, description string, startingPrice int, startDateTime time.Time, endDateTime time.Time, duration time.Duration, auctionService *auction.AuctionHTTP) *Hub {
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
		StartTime:     startDateTime,
		EndTime:       endDateTime,
		auctionHTTP:   *auctionService,
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

	// Update client count in database asynchronously
	go h.updateClientCountInDB(clientCount)

	// Send user joined message to other clients
	userJoinedMsg := types.NewUserJoinedMessage(h.AuctionID, client.ID)
	if data, err := json.Marshal(userJoinedMsg); err == nil {
		h.broadcastToOthers(data, client.ID)
	} else {
		log.Printf("Failed to marshal user joined message: %v", err)
	}

	// Send auction data to the new client
	h.sendAuctionDataToClient(client)

	// Send current bid if exists
	if h.HighestBid != nil {
		currentBidMsg := types.NewBidUpdateMessage(h.AuctionID, h.HighestBid.SenderID, h.HighestBid.Price, &client.UserName)
		if data, err := json.Marshal(currentBidMsg); err == nil {
			h.sendToClientWithTimeout(client, data, RegistrationTimeout)
		} else {
			log.Printf("Failed to marshal current bid message: %v", err)
		}
	}
}

func (h *Hub) handleClientUnregistration(client *Client) {
	h.mu.Lock()
	_, existed := h.Clients[client]
	h.removeClientUnsafe(client)
	clientCount := len(h.Clients)
	h.mu.Unlock()

	if !existed {
		return
	}

	log.Printf("Client %s left auction %s (remaining clients: %d)", client.ID, h.AuctionID, clientCount)

	// Update client count in database asynchronously
	go h.updateClientCountInDB(clientCount)

	userLeftMsg := types.NewUserLeftMessage(h.AuctionID, client.ID)
	if data, err := json.Marshal(userLeftMsg); err == nil {
		h.broadcastToOthers(data, client.ID)
	} else {
		log.Printf("Failed to marshal user left message: %v", err)
	}
}

type BidValidationResult struct {
	IsValid      bool
	ErrorMessage string
	MinRequired  float64
}

func (h *Hub) validateBid(bid *Bid) BidValidationResult {
	if bid == nil {
		return BidValidationResult{
			IsValid:      false,
			ErrorMessage: "Invalid bid data",
		}
	}

	if !h.IsActive || time.Now().After(h.EndTime) {
		return BidValidationResult{
			IsValid:      false,
			ErrorMessage: "Auction has ended",
		}
	}

	if bid.Price <= 0 {
		return BidValidationResult{
			IsValid:      false,
			ErrorMessage: "Bid amount must be positive",
		}
	}

	const maxBidAmount = 10_000_000_000_000
	if bid.Price > maxBidAmount {
		return BidValidationResult{
			IsValid:      false,
			ErrorMessage: "Bid amount too large",
		}
	}

	if h.HighestBid != nil && h.HighestBid.SenderID == bid.SenderID {
		return BidValidationResult{
			IsValid:      false,
			ErrorMessage: "Cannot outbid yourself",
		}
	}

	var minRequired float64
	if h.HighestBid != nil {
		fmt.Println("Highest Bid:", h.HighestBid.Price)
		fmt.Println("Increment:", h.Increment)
		minRequired = h.HighestBid.Price + h.Increment
	} else {
		fmt.Println("Starting Price:", h.StartingPrice)
		minRequired = h.StartingPrice + h.Increment
	}

	if bid.Price < minRequired {
		return BidValidationResult{
			IsValid:      false,
			ErrorMessage: fmt.Sprintf("Bid too low. Minimum required: $%.2f", minRequired),
			MinRequired:  minRequired,
		}
	}

	return BidValidationResult{
		IsValid:     true,
		MinRequired: minRequired,
	}
}

func (h *Hub) handleBid(bid *Bid) {
	log.Printf("Received bid: %+v", bid)

	h.mu.Lock()
	defer h.mu.Unlock()

	log.Printf("[Debug] Bid check for AuctionID=%s | IsActive=%v | EndTime=%v | Now=%v",
		h.AuctionID, h.IsActive, h.EndTime.Format(time.RFC3339), time.Now().Format(time.RFC3339))

	// Validate the bid
	validation := h.validateBid(bid)
	if !validation.IsValid {
		log.Printf("Bid validation failed: %v", validation)
		log.Printf("Bid: %+v", bid)
		log.Printf("SenderID: %s", bid.SenderID)

		h.sendBidRejection(bid.SenderID, validation.ErrorMessage, bid.UserName)
		h.addBidToHistory(bid) // Always add to history for audit purposes
		return
	}

	// Process valid bid
	h.lastActive = time.Now()
	bid.Timestamp = time.Now()
	h.HighestBid = bid
	h.addBidToHistory(bid)

	go func() {
		_, err := h.auctionHTTP.AddBid(h.Ctx, &types.NewBidRequest{
			Amount:    bid.Price,
			SenderID:  bid.SenderID,
			AuctionID: h.AuctionID,
		})
		if err != nil {
			log.Printf("Failed to add bid to auction %s: %v", h.AuctionID, err)
			return
		}
		fmt.Println("Bid Added to Auction")
	}()

	// Broadcast successful bid
	if err := h.broadcastBidUpdate(bid); err != nil {
		log.Printf("Failed to broadcast bid update in auction %s: %v", h.AuctionID, err)
		return
	}

	log.Printf("New highest bid in auction %s: $%.2f by %s", h.AuctionID, bid.Price, bid.SenderID)
}

func (h *Hub) sendBidRejection(bidderID, message string, userName *string) {
	rejectionMsg := types.NewErrorMessage(h.AuctionID, bidderID, message, userName)
	if data, err := json.Marshal(rejectionMsg); err == nil {
		h.sendToBidderUnsafe(data, bidderID)
	} else {
		log.Printf("Failed to marshal bid rejection message: %v", err)
	}
}

func (h *Hub) broadcastBidUpdate(bid *Bid) error {
	bidUpdateMsg := types.NewBidUpdateMessage(h.AuctionID, bid.SenderID, bid.Price, bid.UserName)
	data, err := json.Marshal(bidUpdateMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal bid update message: %w", err)
	}

	h.broadcastUnsafe(data)
	return nil
}

func (h *Hub) addBidToHistory(bid *Bid) {
	if bid != nil {
		h.BidHistory = append(h.BidHistory, bid)
	}
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
			return
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
		CurrentPrice:  h.StartingPrice,
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

	auctionDataMsg := types.NewAuctionDataMessage(h.AuctionID, auctionData, client.ID)
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

	return &types.AuctionData{
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
			JoinedAt: time.Now(), // Note: This should ideally be stored when client joins
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

func (h *Hub) updateClientCountInDB(clientCount int) {
	// Create a context with timeout for the database operation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Update client count in database using the auction service
	if err := h.auctionHTTP.UpdateClientCount(ctx, h.AuctionID, clientCount); err != nil {
		log.Printf("Failed to update client count in database for auction %s: %v", h.AuctionID, err)
	} else {
		log.Printf("Successfully updated client count in database for auction %s: %d", h.AuctionID, clientCount)
	}
}

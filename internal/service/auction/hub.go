package auction

import (
	"context"
	"encoding/json"
	"log"
	"time"

	types "github.com/LikhithMar14/BidZy/pkg/types"
)

type Bid struct {
	SenderID  string    `json:"sender_id"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Bid        chan *Bid
	Unregister chan *Client
	Ctx        context.Context
	Cancel     context.CancelFunc
	HighestBid *Bid
	AuctionID  string
	MinBidIncrement float64
}

func NewHub(auctionID string) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		Clients:         make(map[*Client]bool),
		Broadcast:       make(chan []byte, 256), 
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
		Bid:             make(chan *Bid),
		Ctx:             ctx,
		Cancel:          cancel,
		AuctionID:       auctionID,
		MinBidIncrement: 1.0, 
	}
}

func (h *Hub) Run() {
	defer func() {
		for client := range h.Clients {
			close(client.Send)
		}
	}()

	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			log.Printf("Client %s registered to auction %s", client.ID, h.AuctionID)
			
			if h.HighestBid != nil {
				if response, err := json.Marshal(types.Message{
					Type:         types.TypeBid,
					Action:       types.ActionCurrentBid,
					AuctionID:    h.AuctionID,
					SenderID:     h.HighestBid.SenderID,
					BiddingPrice: h.HighestBid.Price,
					Content:      "Current highest bid",
					Timestamp:    h.HighestBid.Timestamp,
				}); err == nil {
					select {
					case client.Send <- response:
					case <-time.After(time.Second):
						log.Printf("Failed to send current bid to client %s", client.ID)
					}
				}
			}

		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				log.Printf("Client %s unregistered from auction %s", client.ID, h.AuctionID)
			}

		case message := <-h.Broadcast:
			h.broadcastToClients(message)

		case bid := <-h.Bid:
			h.processBid(bid)

		case <-h.Ctx.Done():
			log.Printf("Hub for auction %s shutting down", h.AuctionID)
			return
		}
	}
}

func (h *Hub) processBid(bid *Bid) {
	if bid.Price <= 0 {
		log.Printf("Invalid bid amount: %f from %s", bid.Price, bid.SenderID)
		return
	}

	if h.HighestBid != nil {
		if bid.Price <= h.HighestBid.Price {
			log.Printf("Rejected bid from %s: %f not higher than current %f", 
				bid.SenderID, bid.Price, h.HighestBid.Price)
			h.sendBidRejection(bid.SenderID, "Bid must be higher than current highest bid")
			return
		}
		
		if bid.Price < h.HighestBid.Price + h.MinBidIncrement {
			log.Printf("Rejected bid from %s: increment too small", bid.SenderID)
			h.sendBidRejection(bid.SenderID, "Bid increment too small")
			return
		}
	}
	
	bid.Timestamp = time.Now()
	h.HighestBid = bid
	
	response, err := json.Marshal(types.Message{
		Type:         types.TypeBid,
		Action:       types.ActionPlaceBid,
		AuctionID:    h.AuctionID,
		SenderID:     bid.SenderID,
		BiddingPrice: bid.Price,
		Content:      "New highest bid",
		Timestamp:    bid.Timestamp,
	})
	
	if err != nil {
		log.Printf("Error marshaling bid response: %v", err)
		return
	}

	log.Printf("New highest bid in auction %s: %f from %s", h.AuctionID, bid.Price, bid.SenderID)
	h.broadcastToClients(response)
}

func (h *Hub) sendBidRejection(senderID, reason string) {
		response, err := json.Marshal(types.Message{
		Type:      types.TypeError,
		Action:    types.ActionBidRejected,
		AuctionID: h.AuctionID,
		SenderID:  senderID,
		Content:   reason,
		Timestamp: time.Now(),
	})
	
	if err != nil {
		return
	}

	for client := range h.Clients {
		if client.ID == senderID {
			select {
			case client.Send <- response:
			case <-time.After(time.Second):
				log.Printf("Failed to send rejection to client %s", client.ID)
			}
			break
		}
	}
}

func (h *Hub) broadcastToClients(message []byte) {
	for client := range h.Clients {
		select {
				case client.Send <- message:
				case <-time.After(time.Second):
			log.Printf("Client %s send channel blocked, removing", client.ID)
			close(client.Send)
			delete(h.Clients, client)
		}
	}
}

func (h *Hub) GetClientCount() int {
	return len(h.Clients)
}
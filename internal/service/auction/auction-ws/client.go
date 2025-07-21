package auction_ws

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	types "github.com/LikhithMar14/BidZy/pkg/types"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	sendTimeout    = 5 * time.Second
)

type Client struct {
	ID       string
	UserName string
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan []byte
	Rdb      *redis.Client
}

func (c *Client) ReadPump() {
	defer func() {
		log.Printf("ReadPump for client %s ending", c.ID)
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-c.Hub.Ctx.Done():
			log.Printf("Hub context cancelled for client %s", c.ID)
			return
		default:
			_, messageBytes, err := c.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error for client %s: %v", c.ID, err)
				}
				return
			}

			var msg types.Message
			if err := json.Unmarshal(messageBytes, &msg); err != nil {
				log.Printf("Invalid JSON message from client %s: %v", c.ID, err)
				c.sendErrorMessage("Invalid message format")
				continue
			}

			if !msg.IsValid() {
				log.Printf("Invalid message structure from client %s: %+v", c.ID, msg)
				c.sendErrorMessage("Invalid message structure")
				continue
			}

			if msg.SenderID == "" {
				msg.SenderID = c.ID
			}

			if msg.AuctionID != c.Hub.AuctionID {
				log.Printf("Auction ID mismatch from client %s: expected %s, got %s", c.ID, c.Hub.AuctionID, msg.AuctionID)
				c.sendErrorMessage("Auction ID mismatch")
				continue
			}

			c.handleMessage(&msg)
		}
	}
}

func (c *Client) handleMessage(msg *types.Message) {
	switch msg.Type {
	case types.TypeAuction:
		c.handleAuctionMessage(msg)
	case types.TypeBid:
		c.handleBidMessage(msg)
	case types.TypePing:
		c.handlePingMessage(msg)
	default:
		log.Printf("Unknown message type from client %s: %s", c.ID, msg.Type)
		c.sendErrorMessage("Unknown message type")
	}
}

func (c *Client) handleAuctionMessage(msg *types.Message) {
	switch msg.Action {
	case types.ActionJoin:
		log.Println("I AM HERE IN JOIN ACTION")
		log.Printf("Client %s joined auction %s", msg.SenderID, msg.AuctionID)
		c.sendSuccessMessage("Successfully joined auction")

	case types.ActionLeave:
		log.Printf("Client %s leaving auction %s", msg.SenderID, msg.AuctionID)
		select {
		case c.Hub.Unregister <- c:
		case <-c.Hub.Ctx.Done():
			return
		case <-time.After(ChannelTimeout):
			log.Printf("Timeout unregistering client %s", c.ID)
			return
		}
		return

	case types.ActionGetAuctionData:
		log.Printf("Client %s requested auction data for %s", msg.SenderID, msg.AuctionID)
		c.sendAuctionData()

	case types.ActionCurrentBid:
		log.Printf("Client %s requested current bid for auction %s", msg.SenderID, msg.AuctionID)
		c.sendCurrentBidAndCount()

	default:
		log.Printf("Unknown auction action from client %s: %s", c.ID, msg.Action)
		c.sendErrorMessage("Unknown auction action")
	}
}

func (c *Client) handleBidMessage(msg *types.Message) {
	// Validate bid input
	if msg.Action != types.ActionPlaceBid {
		log.Printf("Invalid bid action from client %s: %s", c.ID, msg.Action)
		c.sendErrorMessage("Invalid bid action")
		return
	}

	if msg.BiddingPrice <= 0 {
		log.Printf("Invalid bid price from client %s: %f", c.ID, msg.BiddingPrice)
		c.sendErrorMessage("Bid price must be positive")
		return
	}

	if msg.BiddingPrice > 10_000_000_000_000 {
		log.Printf("Bid price too large from client %s: %f", c.ID, msg.BiddingPrice)
		c.sendErrorMessage("Bid price too large")
		return
	}

	bid := &Bid{
		SenderID: msg.SenderID,
		Price:    msg.BiddingPrice,
		UserName: &c.UserName,
	}

	select {
	case c.Hub.Bid <- bid:
		log.Printf("Bid submitted by %s: $%.2f", msg.SenderID, msg.BiddingPrice)
	case <-c.Hub.Ctx.Done():
		return
	case <-time.After(ChannelTimeout):
		log.Printf("Timeout submitting bid from client %s", c.ID)
		c.sendErrorMessage("Bid submission timeout")
	}
}

func (c *Client) handlePingMessage(msg *types.Message) {
	pongMsg := types.NewPongMessage(msg.AuctionID, &c.UserName)
	pongMsg.SenderID = c.ID

	if data, err := json.Marshal(pongMsg); err == nil {
		select {
		case c.Send <- data:
		case <-c.Hub.Ctx.Done():
			return
		case <-time.After(sendTimeout):
			log.Printf("Timeout sending pong to client %s", c.ID)
			return
		}
	} else {
		log.Printf("Failed to marshal pong message for client %s: %v", c.ID, err)
	}
}

func (c *Client) sendAuctionData() {
	auctionData := c.Hub.GetAuctionData()
	auctionDataMsg := types.NewAuctionDataMessage(c.Hub.AuctionID, auctionData, c.ID)

	if data, err := json.Marshal(auctionDataMsg); err == nil {
		select {
		case c.Send <- data:
		case <-time.After(sendTimeout):
			log.Printf("Timeout sending auction data to client %s", c.ID)
		}
	} else {
		log.Printf("Failed to marshal auction data for client %s: %v", c.ID, err)
	}
}

func (c *Client) sendCurrentBidAndCount() {

	if c.Hub.HighestBid != nil {
		bidUpdateMsg := types.NewBidUpdateMessage(c.Hub.AuctionID, c.Hub.HighestBid.SenderID, c.Hub.HighestBid.Price, &c.UserName)
		if data, err := json.Marshal(bidUpdateMsg); err == nil {
			select {
			case c.Send <- data:
			case <-time.After(sendTimeout):
				log.Printf("Timeout sending current bid to client %s", c.ID)
			}
		}
	}

	countMsg := types.NewCountMessage(c.Hub.AuctionID, c.Hub.GetClientCount())
	if data, err := json.Marshal(countMsg); err == nil {
		select {
		case c.Send <- data:
		case <-time.After(sendTimeout):
			log.Printf("Timeout sending count to client %s", c.ID)
		}
	} else {
		log.Printf("Failed to marshal count message: %v", err)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
		log.Printf("WritePump for client %s ending", c.ID)
	}()

	for {
		select {
		case <-c.Hub.Ctx.Done():
			log.Printf("Hub context cancelled for client %s write pump", c.ID)
			return

		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write error for client %s: %v", c.ID, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error for client %s: %v", c.ID, err)
				return
			}
		}
	}
}

func (c *Client) sendErrorMessage(content string) {
	errorMsg := types.NewErrorMessage(c.Hub.AuctionID, c.ID, content, &c.UserName)
	errorMsg.SenderID = c.ID

	data, err := json.Marshal(errorMsg)
	if err != nil {
		log.Printf("Failed to marshal error message for client %s: %v", c.ID, err)
		return
	}

	select {
	case c.Send <- data:
	case <-time.After(sendTimeout):
		log.Printf("Timeout sending error message to client %s", c.ID)
	}
}

func (c *Client) sendSuccessMessage(content string) {
	fmt.Println("Sending success message to client", c)
	successMsg := types.NewSuccessMessage(c.Hub.AuctionID, content, &c.UserName)
	successMsg.SenderID = c.ID
	successMsg.UserName = &c.UserName

	data, err := json.Marshal(successMsg)
	fmt.Println("Success message", successMsg)
	if err != nil {
		log.Printf("Failed to marshal success message for client %s: %v", c.ID, err)
		return
	}

	select {
	//if it is in case of redis , then we publish the message to the redis channel
	case c.Send <- data:
	case <-time.After(sendTimeout):
		log.Printf("Timeout sending success message to client %s", c.ID)
	}
}

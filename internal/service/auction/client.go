package auction

import (
	"encoding/json"
	"log"
	"time"

	types "github.com/LikhithMar14/BidZy/pkg/types"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Client struct {
	ID    string
	Hub   *Hub
	Conn  *websocket.Conn
	Send  chan []byte

}


func (c *Client) ReadPump() {
	defer func() {
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
				log.Printf("Invalid message from client %s: %v", c.ID, err)
				continue
			}

			if msg.SenderID == "" {
				msg.SenderID = c.ID
			}

			switch msg.Type {
			case types.TypeBid:
				if msg.Action == types.ActionPlaceBid && msg.BiddingPrice > 0 {
					c.Hub.Bid <- &Bid{
						SenderID: msg.SenderID,
						Price:    msg.BiddingPrice,
					}
				} else {
					log.Printf("Invalid bid from client %s: price %f", c.ID, msg.BiddingPrice)
				}
			case types.TypePing:
				pong, _ := json.Marshal(types.Message{
					Type:      types.TypePong,
					AuctionID: msg.AuctionID,
					SenderID:  c.ID,
					Timestamp: time.Now(),
				})
				select {
				case c.Send <- pong:
				default:
					return
				}
			default:
				log.Printf("Unhandled message type %s from client %s", msg.Type, c.ID)
			}
		}
	}
}

func (c *Client) WritePump() {

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		select {
        case <-c.Send:
        default:
            close(c.Send)
        }
	}()

	for {
		select {
		case <-c.Hub.Ctx.Done():
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
				return
			}
		}
	}
}

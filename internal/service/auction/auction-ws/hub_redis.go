package auction_ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/LikhithMar14/BidZy/internal/service/auction"
	"github.com/LikhithMar14/BidZy/pkg/types"
	"github.com/redis/go-redis/v9"
)



type EventPayload struct {
	Type string `json:"type"`
	ClientID string `json:"client_id"`
	Data json.RawMessage `json:"data"`
}
type HubRedis struct {
	AuctionID     string
	Title         string
	Description   string
	StartingPrice float64
	Increment     float64
	StartTime     time.Time
	EndTime       time.Time
	auctionHTTP   auction.AuctionHTTP
	Rdb           *redis.Client
	Ctx           context.Context
	cancel        context.CancelFunc
}

func NewHubRedis(
	ctx context.Context,
	rdb *redis.Client,
	auctionID string,
	title string,
	description string,
	startingPrice float64,
	increment float64,
	startTime, endTime time.Time,
	auctionService *auction.AuctionHTTP,
) *HubRedis {
	c, cancel := context.WithCancel(ctx)

	return &HubRedis{
		AuctionID:     auctionID,
		Title:         title,
		Description:   description,
		StartingPrice: startingPrice,
		Increment:     increment,
		StartTime:     startTime,
		EndTime:       endTime,
		auctionHTTP:   *auctionService,
		Rdb:           rdb,
		Ctx:           c,
		cancel:        cancel,
	}
}


func (h *HubRedis) registerClientRedis(clientID string) error {
	key := fmt.Sprintf("auction:%s:clients",h.AuctionID)
	if err := h.Rdb.SIsMember(h.Ctx, key, clientID).Err(); err == nil {
		return fmt.Errorf("client %s already registered", clientID)
	}	
	err := h.Rdb.SAdd(h.Ctx, key, clientID).Err()
	if err != nil {
		return err
	}
	return nil
}

func (h *HubRedis) unregisterClientRedis(clientID string) error {
	key := fmt.Sprintf("auction:%s:clients",h.AuctionID)
	if err := h.Rdb.SIsMember(h.Ctx, key, clientID).Err(); err != nil {
		return fmt.Errorf("client %s not registered", clientID)
	}
	err := h.Rdb.SRem(h.Ctx, key, clientID).Err()
	if err != nil {
		return err
	}
	return nil
}

func (h *HubRedis) closeAllClientsRedis() error {
	channel := fmt.Sprintf("auction:%s:control", h.AuctionID)


	err := h.Rdb.Publish(h.Ctx, channel, map[string]string {
		"type" : "shutdown",
	}).Err()
	if err != nil {
		log.Printf("HubRedis: %s error publishing shutdown message: %v", h.AuctionID, err)
		return err
	}
	//for futrue refernce take care of storing things to database also

	h.Rdb.Del(h.Ctx, fmt.Sprintf("auction:%s:clients", h.AuctionID))

	return nil

	
}
func (h *HubRedis) Run() error {
	defer func() {
		log.Printf("HubRedis: %s shutting down", h.AuctionID)
		if err := h.closeAllClientsRedis(); err != nil {
			log.Printf("HubRedis: error closing clients: %v", err)
		}
	}()

	// Subscribe to main auction channel and control channel
	mainChannel := fmt.Sprintf("auction:%s", h.AuctionID)
	controlChannel := fmt.Sprintf("auction:%s:control", h.AuctionID)

	pubsub := h.Rdb.Subscribe(h.Ctx, mainChannel, controlChannel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case <-h.Ctx.Done():
			log.Printf("HubRedis: %s context cancelled", h.AuctionID)
			return nil

		case msg := <-ch:
			if msg.Channel == controlChannel {
				if msg.Payload == `{"type":"shutdown"}` {
					log.Printf("HubRedis: %s received shutdown signal", h.AuctionID)
					return nil
				}
			}

			var event EventPayload
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Printf("HubRedis: %s error unmarshalling event: %v", h.AuctionID, err)
				continue
			}

			switch event.Type {
			case string(types.ActionJoin):
				err := h.registerClientRedis(event.ClientID)
				if err != nil {
					log.Printf("Join Error: %v", err)
				}
				h.broadcastUserJoinedRedis(event.ClientID)

			case string(types.ActionLeave):
				err := h.unregisterClientRedis(event.ClientID)
				if err != nil {
					log.Printf("Leave Error: %v", err)
				}
				h.broadcastUserLeftRedis(event.ClientID)

			case string(types.ActionPlaceBid):
				h.handlePlaceBidRedis(event.ClientID, event.Data)

			default:
				log.Printf("HubRedis: unknown action type received: %s", event.Type)
			}
		}
	}
}

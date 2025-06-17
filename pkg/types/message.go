package types

// {
// 	"type": "auction",
// 	"action": "leave",
// 	"auctionId":"b6e8efe7-a1bc-4bed-b53a-c06428566b12",
// 	"senderId":"edc4547c-a191-4b02-8f49-9dda4c8854b"
//   }

// {
// 	"type": "bid",
// 	"action": "place_bid",
// 	"auctionId": "b6e8efe7-a1bc-4bed-b53a-c06428566b12",
// 	"senderId": "0bbddc41-cd43-4660-9ceb-876b0e678769",
// 	"biddingPrice": 350
//   }

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type MessageType string

const (
	TypeAuction     MessageType = "auction"
	TypeBid         MessageType = "bid"
	TypeError       MessageType = "error"
	TypePing        MessageType = "ping"
	TypePong        MessageType = "pong"
	TypeCount       MessageType = "count"
	TypeAuctionData MessageType = "auction_data"
	TypeUserJoined  MessageType = "user_joined"
	TypeUserLeft    MessageType = "user_left"
	TypeBidUpdate   MessageType = "bid_update"
	TypeSuccess     MessageType = "success"
)

type AuctionAction string

const (
	ActionJoin           AuctionAction = "join"
	ActionLeave          AuctionAction = "leave"
	ActionPlaceBid       AuctionAction = "place_bid"
	ActionCurrentBid     AuctionAction = "current_bid"
	ActionBidRejected    AuctionAction = "bid_rejected"
	ActionBidAccepted    AuctionAction = "bid_accepted"
	ActionGetAuctionData AuctionAction = "get_auction_data"
	ActionAuctionStarted AuctionAction = "auction_started"
	ActionAuctionEnded   AuctionAction = "auction_ended"
)

type Message struct {
	Type         MessageType   `json:"type"`
	Action       AuctionAction `json:"action,omitempty"`
	AuctionID    string        `json:"auctionId"`
	SenderID     string        `json:"senderId"`
	BiddingPrice float64       `json:"biddingPrice,omitempty"`
	Content      string        `json:"content,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
	Count        int           `json:"count,omitempty"`
	Success      bool          `json:"success,omitempty"`
	Data         interface{}   `json:"data,omitempty"`
	UserName     *string       `json:"userName,omitempty"`
}

type WebSocketMessage struct {
	Type         MessageType   `json:"type"`
	Action       AuctionAction `json:"action"`
	AuctionID    *string       `json:"auctionId,omitempty"`
	SenderID     *string       `json:"senderId,omitempty"`
	BiddingPrice float64       `json:"biddingPrice,omitempty"`
	Content      string        `json:"content,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
}

type AuctionData struct {
	AuctionID     string    `json:"auctionId"`
	Title         string    `json:"title,omitempty"`
	Description   string    `json:"description,omitempty"`
	StartingPrice float64   `json:"startingPrice,omitempty"`
	CurrentPrice  float64   `json:"currentPrice"`
	HighestBidder string    `json:"highestBidder,omitempty"`
	ClientCount   int       `json:"clientCount"`
	IsActive      bool      `json:"isActive"`
	Status        string    `json:"status"`
	StartTime     time.Time `json:"startTime,omitempty"`
	EndTime       time.Time `json:"endTime,omitempty"`
	Increment     float64   `json:"increment"`
	Image         string    `json:"image"`
	User          User      `json:"user"`
	CategoryIDs   []int     `json:"categoryIds"`
	Participants  []User    `json:"participants"`
}

func (m *Message) IsValid() bool {
	if m.Type == "" || m.AuctionID == "" {
		return false
	}

	switch m.Type {
	case TypeAuction:
		if m.SenderID == "" {
			return false
		}
		validActions := []AuctionAction{ActionJoin, ActionLeave, ActionCurrentBid, ActionBidRejected, ActionGetAuctionData, ActionAuctionStarted, ActionAuctionEnded}
		return slices.Contains(validActions, m.Action)
	case TypeBid:
		return m.Action == ActionPlaceBid && m.BiddingPrice > 0 && m.SenderID != "" && m.AuctionID != ""
	case TypePing, TypePong, TypeCount, TypeUserJoined, TypeUserLeft, TypeBidUpdate, TypeSuccess:
		return true
	case TypeError:
		return m.Content != ""
	case TypeAuctionData:
		return true
	default:
		return false
	}
}

func NewAuctionMessage(action AuctionAction, auctionID, senderID string, userName *string) *Message {
	return &Message{
		Type:      TypeAuction,
		Action:    action,
		AuctionID: auctionID,
		SenderID:  senderID,
		UserName:  userName,
		Timestamp: time.Now(),
	}
}

func NewBidMessage(auctionID, senderID string, price float64, userName *string) *Message {
	return &Message{
		Type:         TypeBid,
		Action:       ActionPlaceBid,
		AuctionID:    auctionID,
		SenderID:     senderID,
		UserName:     userName,
		BiddingPrice: price,
		Timestamp:    time.Now(),
	}
}

func NewErrorMessage(auctionID, senderID, content string, userName *string) *Message {
	return &Message{
		Type:      TypeError,
		AuctionID: auctionID,
		SenderID:  senderID,
		Content:   content,
		UserName:  userName,
		Timestamp: time.Now(),
	}
}

func NewPingMessage(auctionID string, userName *string) *Message {
	return &Message{
		Type:      TypePing,
		AuctionID: auctionID,
		UserName:  userName,
		Timestamp: time.Now(),
	}
}

func NewPongMessage(auctionID string, userName *string) *Message {
	return &Message{
		Type:      TypePong,
		AuctionID: auctionID,
		Timestamp: time.Now(),
		UserName:  userName,
	}
}

func NewCountMessage(auctionID string, count int) *Message {
	return &Message{
		Type:      TypeCount,
		AuctionID: auctionID,
		Count:     count,
		Timestamp: time.Now(),
	}
}

func NewSuccessMessage(auctionID, content string, userName *string) *Message {
	return &Message{
		Type:      TypeSuccess,
		AuctionID: auctionID,
		Content:   content,
		UserName:  userName,
		Success:   true,
		Timestamp: time.Now(),
	}
}

func NewUserJoinedMessage(auctionID, userID string) *Message {
	fmt.Println("NewUserJoinedMessage", auctionID, userID)
	return &Message{
		Type:      TypeUserJoined,
		AuctionID: auctionID,
		SenderID:  strings.TrimSpace(userID),
		Content:   userID + " joined the auction",
		Timestamp: time.Now(),
	}
}

func NewUserLeftMessage(auctionID, userID string) *Message {
	return &Message{
		Type:      TypeUserLeft,
		AuctionID: auctionID,
		SenderID:  userID,
		Content:   userID + " left the auction",
		Timestamp: time.Now(),
	}
}

func NewBidUpdateMessage(auctionID, bidderID string, price float64, userName *string) *Message {
	fmt.Println("NewBidUpdateMessage", auctionID, bidderID, price, *userName)
	return &Message{
		Type:         TypeBidUpdate,
		Action:       ActionBidAccepted,
		AuctionID:    auctionID,
		SenderID:     bidderID,
		BiddingPrice: price,
		Timestamp:    time.Now(),
		UserName:     userName,
	}
}

func NewAuctionDataMessage(auctionID string, data *AuctionData, senderID string) *Message {
	fmt.Println("NewAuctionDataMessage", auctionID, data)
	fmt.Println("SENDING AUCTION DATA TO CLIENT")
	fmt.Println("DATA", data)
	return &Message{
		Type:      TypeAuctionData,
		AuctionID: auctionID,
		SenderID:  senderID,
		Data:      data,
		Timestamp: time.Now(),
	}
}

type CreateAuctionRequest struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	StartingPrice float64   `json:"startingPrice"`
	Increment     float64   `json:"increment"`
	Duration      int       `json:"durationHours"`
	UserID        string    `json:"userId"`
	Image         string    `json:"image"`
	CategoryIDs   []int     `json:"categoryIds"`
	StartDateTime time.Time `json:"startDateTime"`
	EndDateTime   time.Time `json:"endDateTime"`
	Status        string    `json:"status"`
}

type NewBidRequest struct {
	Amount    float64 `json:"amount"`
	SenderID  string  `json:"senderId"`
	AuctionID string  `json:"auctionId"`
}

type NewBidResponse struct {
	ID        string    `json:"id"`
	Amount    float64   `json:"amount"`
	CreatedAt time.Time `json:"createdAt"`
}

type Bid struct {
	ID         string    `json:"id"`
	Amount     float64   `json:"amount"`
	CreatedAt  time.Time `json:"createdAt"`
	SenderID   string    `json:"senderId"`
	AuctionID  string    `json:"auctionId"`
	BidderName string    `json:"bidderName"`
}
type Auction struct {
	ID            string
	Title         string
	Description   string
	StartingPrice float64
	CurrentPrice  float64
	Increment     float64
	StartDate     time.Time
	EndDate       time.Time
	Status        string
	Image         string
	ClientCount   int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type BidTimelineEntry struct {
	BidID     string    `json:"bid_id"`
	Amount    float64   `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
	Bidder    string    `json:"bidder"`
}

type CreateAuctionResponse struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	StartingPrice float64   `json:"startingPrice"`
	CurrentPrice  float64   `json:"currentPrice"`
	StartDateTime time.Time `json:"startDateTime"`
	EndDateTime   time.Time `json:"endDateTime"`
	Status        string    `json:"status"`
	Image         string    `json:"image"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	User          User      `json:"user"`
	CategoryIDs   []int     `json:"categoryIds"`
	Increment     float64   `json:"increment"`
}

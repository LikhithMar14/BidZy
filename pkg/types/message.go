package types

import "slices"

import "time"

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
	StartTime     time.Time `json:"startTime,omitempty"`
	EndTime       time.Time `json:"endTime,omitempty"`
	Increment     float64   `json:"increment"`
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

func NewAuctionMessage(action AuctionAction, auctionID, senderID string) *Message {
	return &Message{
		Type:      TypeAuction,
		Action:    action,
		AuctionID: auctionID,
		SenderID:  senderID,
		Timestamp: time.Now(),
	}
}

func NewBidMessage(auctionID, senderID string, price float64) *Message {
	return &Message{
		Type:         TypeBid,
		Action:       ActionPlaceBid,
		AuctionID:    auctionID,
		SenderID:     senderID,
		BiddingPrice: price,
		Timestamp:    time.Now(),
	}
}

func NewErrorMessage(auctionID, content string) *Message {
	return &Message{
		Type:      TypeError,
		AuctionID: auctionID,
		Content:   content,
		Timestamp: time.Now(),
	}
}

func NewPingMessage(auctionID string) *Message {
	return &Message{
		Type:      TypePing,
		AuctionID: auctionID,
		Timestamp: time.Now(),
	}
}

func NewPongMessage(auctionID string) *Message {
	return &Message{
		Type:      TypePong,
		AuctionID: auctionID,
		Timestamp: time.Now(),
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

func NewSuccessMessage(auctionID, content string) *Message {
	return &Message{
		Type:      TypeSuccess,
		AuctionID: auctionID,
		Content:   content,
		Success:   true,
		Timestamp: time.Now(),
	}
}

func NewUserJoinedMessage(auctionID, userID string) *Message {
	return &Message{
		Type:      TypeUserJoined,
		AuctionID: auctionID,
		SenderID:  userID,
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

func NewBidUpdateMessage(auctionID, bidderID string, price float64) *Message {
	return &Message{
		Type:         TypeBidUpdate,
		Action:       ActionBidAccepted,
		AuctionID:    auctionID,
		SenderID:     bidderID,
		BiddingPrice: price,
		Timestamp:    time.Now(),
	}
}

func NewAuctionDataMessage(auctionID string, data *AuctionData) *Message {
	return &Message{
		Type:      TypeAuctionData,
		AuctionID: auctionID,
		Data:      data,
		Timestamp: time.Now(),
	}
}


type CreateAuctionRequest struct {
	ID string `json:"id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	StartingPrice float64 `json:"startingPrice"`
	Increment     float64 `json:"increment"`
	Duration      int     `json:"durationHours"` 
	UserID        string  `json:"userId"`
	Image         string  `json:"image"`
	CategoryIDs    []int     `json:"categoryIds"`
	StartDateTime time.Time `json:"startDateTime"`
	EndDateTime   time.Time `json:"endDateTime"`
	Status        string  `json:"status"`


}

type NewBidRequest struct {
	Amount float64 `json:"amount"`
	SenderID string `json:"senderId"`
	AuctionID string `json:"auctionId"`
}

type NewBidResponse struct {
	ID string `json:"id"`
	Amount float64 `json:"amount"`
	CreatedAt time.Time `json:"createdAt"`
}

type Bid struct {
	ID string `json:"id"`
	Amount float64 `json:"amount"`
	CreatedAt time.Time `json:"createdAt"`
	SenderID string `json:"senderId"`
	AuctionID string `json:"auctionId"`
	BidderName string `json:"bidderName"`
	AuctionTitle string `json:"auctionTitle"`
	AuctionStatus string `json:"auctionStatus"`
	AuctionEndDate time.Time `json:"auctionEndDate"`
	AuctionImage string `json:"auctionImage"`

}

type CreateAuctionResponse struct {
	ID string `json:"id"`
	Title string `json:"title"`
	Description string `json:"description"`
	StartingPrice float64 `json:"startingPrice"`
	CurrentPrice float64 `json:"currentPrice"`
	StartDateTime time.Time `json:"startDateTime"`
	EndDateTime time.Time `json:"endDateTime"`
	Status string `json:"status"`
	Image string `json:"image"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	User User `json:"user"`
	CategoryIDs []int `json:"categoryIds"`
	Increment float64 `json:"increment"`
	
}
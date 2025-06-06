package models

import (
	"time"
)

type AuctionStatus int

const (
	ACTIVE AuctionStatus = iota
	ENDED
	CANCELLED
	INACTIVE
)


type AuctionRequest struct {
	Title         string    `json:"title" validate:"required,min=1,max=50"`
	Description   string    `json:"description" validate:"required,min=1"`
	StartingPrice float64   `json:"startingPrice" validate:"required,gt=0"`
	StartDate     time.Time `json:"startDate" validate:"required,gtfield=Now"`
	EndDate       time.Time `json:"endDate" validate:"required,gtfield=StartDate"`
	Categories    string    `json:"categories" validate:"required,oneof=COLLECTABLES WATCHES FASHION ART ELECTRONICS VEHICLES REALESTATE FURNITURE MISCELLANEOUS"`
	Image         string    `json:"image" validate:"required,url"`
	UserID        string    `json:"userId" validate:"required,uuid4"`
}

type BidRequest struct {
	Amount float64 `json:"amount" validate:"required,gt=0"`
	AuctionID string `json:"auctionId" validate:"required,uuid4"`
	UserID    string `json:"userId" validate:"required,uuid4"`
}

type Bid struct {
	ID        string    `json:"id" validate:"required,uuid4"`
	Amount    float64   `json:"amount" validate:"required,gt=0"`
	CreatedAt time.Time `json:"createdAt" validate:"required"`
	UserID    string    `json:"userId" validate:"required,uuid4"`
	AuctionID string    `json:"auctionId" validate:"required,uuid4"`
}

type User struct {
	ID             string    `json:"id" validate:"required,uuid4"`
	UserName       string    `json:"userName" validate:"required"`
	Email          string    `json:"email" validate:"required,email"`
	HashedPassword string    `json:"hashedPassword" validate:"required"`
	CreatedAt      time.Time `json:"createdAt" validate:"required"`
	UpdatedAt      time.Time `json:"updatedAt" validate:"required"`
}

type Auction struct {
	ID           string       `json:"id" validate:"required,uuid4"`
	Title        string       `json:"title" validate:"required,min=1,max=50"`
	Description  string       `json:"description" validate:"required"`
	StartingPrice float64     `json:"startingPrice" validate:"required,gt=0"`
	CurrentPrice float64      `json:"currentPrice" validate:"required,gt=0"`
	StartDate    time.Time    `json:"startDate" validate:"required"`
	EndDate      time.Time    `json:"endDate" validate:"required,gtfield=StartDate"`
	Status       string       `json:"status" validate:"required,oneof=INACTIVE ACTIVE ENDED CANCELLED"`
	CreatedAt    time.Time    `json:"createdAt" validate:"required"`
	UpdatedAt    time.Time    `json:"updatedAt" validate:"required"`
	UserID       string       `json:"userId" validate:"required,uuid4"`
	Image        string       `json:"image" validate:"required,url"`
	Categories   string       `json:"categories" validate:"required,oneof=COLLECTABLES WATCHES FASHION ART ELECTRONICS VEHICLES REALESTATE FURNITURE MISCELLANEOUS"`
	Bids         []Bid        `json:"bids"`
	User         User         `json:"user"`
}




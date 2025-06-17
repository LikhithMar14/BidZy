package auction_ws

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"strconv"
// 	"time"

// 	"github.com/LikhithMar14/BidZy/internal/service/auction"
// 	"github.com/redis/go-redis/v9"
// )

// type HubManagerRedis struct {
// 	rdb *redis.Client
// 	ctx context.Context
// 	cancel context.CancelFunc
// 	stopped bool
// }

// func NewHubManagerRedis(rdb *redis.Client) *HubManagerRedis {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	return &HubManagerRedis {
// 		rdb: rdb,
// 		ctx:    ctx,
// 		cancel: cancel,
// 		stopped: false,
// 	}

// }
// func (m *HubManagerRedis) GetHub(auctionId string) *HubRedis {
// 	metaKey := fmt.Sprintf("auction:%s:meta", auctionId)
// 	meta, err := m.rdb.HGetAll(m.ctx, metaKey).Result()
// 	if err != nil || len(meta) == 0 {
// 		log.Printf("No metadata found for auction %s: %v", auctionId, err)
// 		return nil
// 	}

// 	startPrice, _ := strconv.ParseFloat(meta["startingPrice"], 64)
// 	increment, _ := strconv.ParseFloat(meta["increment"], 64)
// 	startTime, _ := time.Parse(time.RFC3339, meta["startTime"])
// 	endTime, _ := time.Parse(time.RFC3339, meta["endTime"])

// 	ctx, cancel := context.WithCancel(m.ctx)

// 	hub := &HubRedis{
// 		AuctionID:     auctionId,
// 		Title:         meta["title"],
// 		Description:   meta["description"],
// 		StartingPrice: startPrice,
// 		Increment:     increment,
// 		StartTime:     startTime,
// 		EndTime:       endTime,
// 		Ctx:           ctx,
// 		cancel:        cancel,
// 	}

// 	return hub
// }

// func (m *HubManagerRedis) GetOrCreateHub(auctionId string, increment float64, title , description string , startPrice float64, startDateTime time.Time, endDateTime time.Time, duration time.Time, auctionService *auction.AuctionHTTP) *HubRedis{

// 	hub := m.GetHub(auctionId)

// 	if hub != nil {
// 		fmt.Println("Hub already exists for auction %s", auctionId)
// 		return hub
// 	}

// 	hubRedis := NewHubRedis(m.ctx, m.rdb, auctionId, title, description, startPrice, increment, startDateTime, endDateTime, auctionService)

// 	go hubRedis.Run()

// 	return hubRedis
// }

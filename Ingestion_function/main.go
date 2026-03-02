package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/jackc/pgx/v5/pgxpool"
)

var dbPool *pgxpool.Pool

func init() {
	var err error
	dbPool, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
	}
}

type Event struct {
	EventName  string                 `json:"event_name"`
	Timestamp  time.Time              `json:"timestamp"`
	Properties map[string]interface{} `json:"properties"`
	Id string 						  `json:"id"`
	TenantId string 				  `json:"tenant_id"`
}

type PubSubMessage struct {
	Data []byte `json:"data"`
}

// ProcessLogEntry processes a Pub/Sub message from Cloud Logging.
func ProcessMessage(ctx context.Context, m PubSubMessage) error {
	var event Event
	if err := json.Unmarshal(m.Data, &event); err != nil {
		return fmt.Errorf("json.Unmarshal: %v", err)
	}

	query := `
		INSERT INTO events (id, event_name, timestamp, properties,tenantId)
		VALUES ($1, $2, $3, $4,$5)
		ON CONFLICT (id) DO NOTHING`

	_, err := dbPool.Exec(ctx, query, 
		event.Id, 
		event.EventName, 
		event.Timestamp, 
		event.Properties,
		event.TenantId,
	)
	if err != nil {
		return fmt.Errorf("dbPool.Exec: %v", err)
	}

	log.Printf("Successfully ingested event: %s for tenant: ", event.Id)
	return nil

}

func main() {
    ctx := context.Background()
    // Register the function with the framework
    if err := funcframework.RegisterEventFunctionContext(ctx, "/", ProcessMessage); err != nil {
        log.Fatalf("funcframework.RegisterEventFunctionContext: %v\n", err)
    }
    port := "8080"
    if err := funcframework.Start(port); err != nil {
        log.Fatalf("funcframework.Start: %v\n", err)
    }
}
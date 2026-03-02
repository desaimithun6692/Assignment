package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/google/uuid"
	"cloud.google.com/go/pubsub/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)


type DBQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

var db DBQuerier

var dbPool *pgxpool.Pool
var publisher *pubsub.Publisher

func main() {
	ctx := context.Background()
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Initialize the real pool
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	db = pool

	mux := http.NewServeMux()
	
	mux.Handle("/metrics",authMiddleware(http.HandlerFunc(getMetricsHandler)))
	mux.Handle("/events",authMiddleware(http.HandlerFunc(publishEventHandler)))
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}


	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
    topicID := "events-topic"

	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	 if err != nil {
        log.Fatalf("Failed to create pubsub client: %v", err)
    }
	defer pubsubClient.Close()
	publisher = pubsubClient.Publisher(topicID)

	log.Printf("Metrics service starting on :%s...", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func getMetricsHandler(w http.ResponseWriter, r *http.Request) {
	eventName := r.URL.Query().Get("event_name")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if eventName == "" {
		http.Error(w, "event_name query param is required", http.StatusBadRequest)
		return
	}

	query := "SELECT COUNT(*) FROM events WHERE event_name = $1 AND tenant_id = $2"
	tenantID := r.Context().Value(TenantKey).(string)
	args := []interface{}{eventName,tenantID}

	if from != "" {
		query += fmt.Sprintf(" AND timestamp >= $%d", len(args)+1)
		args = append(args, from)
	}
	if to != "" {
		query += fmt.Sprintf(" AND timestamp <= $%d", len(args)+1)
		args = append(args, to)
	}

	var count int64
	err := db.QueryRow(r.Context(), query, args...).Scan(&count)
	if err != nil {
		http.Error(w, "Query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"event_name": eventName,
		"count":      count,
	})
}


type Event struct {
	EventName  string                 `json:"event_name"`
	Timestamp  time.Time              `json:"timestamp"`
	Properties map[string]interface{} `json:"properties"`
	Id string 						  `json:"id"`
	TenantId string 				  `json:"tenant_id"`
}



func publishEventHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

	var event Event

	
    
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

	event.Id = uuid.New().String()
	tenantID := r.Context().Value(TenantKey).(string)
	event.TenantId = tenantID

    // Convert payload to bytes for Pub/Sub
    data, _ := json.Marshal(event)
    
    // Publish returns a 'Result' which can be used to check success
    result := publisher.Publish(r.Context(), &pubsub.Message{
        Data: data,
    })

    // Block until the message is sent or an error occurs
    id, err := result.Get(r.Context())
    if err != nil {
        http.Error(w, "Failed to publish: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]string{"message_id": id})
}


type contextKey string
const TenantKey contextKey = "tenantID"

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // In production, extract this from a verified JWT token
        tenantID := r.Header.Get("X-Tenant-ID")
        if tenantID == "" {
            http.Error(w, "Missing tenant ID", http.StatusUnauthorized)
            return
        }

        // Add tenantID to request context
        ctx := context.WithValue(r.Context(), TenantKey, tenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

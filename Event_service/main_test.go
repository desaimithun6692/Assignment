package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/jackc/pgx/v5"
)

type mockDB struct {
	count int64
}

type mockRow struct {
	count int64
}

func (r *mockRow) Scan(dest ...any) error {
	// We expect the first destination to be the pointer to our count (int64)
	if len(dest) > 0 {
		if ptr, ok := dest[0].(*int64); ok {
			*ptr = r.count
			return nil
		}
	}
	return pgx.ErrNoRows
}


// Implement the interface for our mock
func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return &mockRow{count: m.count}
}



func TestGetMetricsHandler(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		mockCount  int64
		wantStatus int
	}{
		{
			name:       "Success with event_name",
			url:        "/metrics?event_name=page_view",
			mockCount:  100,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Fail missing event_name",
			url:        "/metrics",
			mockCount:  0,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Inject the mock into the global 'db' variable
			db = &mockDB{count: tt.mockCount}

			// 2. Create the request
			req, _ := http.NewRequest("GET", tt.url, nil)
			rr := httptest.NewRecorder()
			
			// 3. Call the handler
			getMetricsHandler(rr, req)

			// 4. Assertions
			if rr.Code != tt.wantStatus {
				t.Errorf("%s: expected status %d, got %d", tt.name, tt.wantStatus, rr.Code)
			}
		})
	}
}
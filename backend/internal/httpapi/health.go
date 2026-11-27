package httpapi

import (
	"context"
	"net/http"
	"time"
)

type healthPinger interface {
	Ping(ctx context.Context) error
}

var healthStore healthPinger

func SetHealthStore(store healthPinger) {
	healthStore = store
}

func HealthLive(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func HealthReady(w http.ResponseWriter, r *http.Request) {
	checkHealth(w, r, true)
}

func Health(w http.ResponseWriter, r *http.Request) {
	checkHealth(w, r, false)
}

func checkHealth(w http.ResponseWriter, r *http.Request, requireDB bool) {
	if healthStore == nil {
		if requireDB {
			writeError(w, http.StatusServiceUnavailable, "not_ready", "Database belum tersedia")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := healthStore.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database_unavailable", "Database tidak tersedia")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

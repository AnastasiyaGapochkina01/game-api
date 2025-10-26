package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	rdb           *redis.Client
	startTime     = time.Now()
	requestCount  uint64
	ctx           = context.Background()
)

type Character struct {
	Name     string `json:"name"`
	Class    string `json:"class"`
	Level    int    `json:"level"`
}

func main() {
	redisAddr := getEnv("REDIS_ADDR", "redis:6379")
	rdb = redis.NewClient(&redis.Options{Addr: redisAddr})

        http.HandleFunc("/", indexHandler)
	http.HandleFunc("/create", createHandler)
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/health", healthHandler)

	addr := ":8080"
	log.Printf("Game API listening on %s", addr)
	http.ListenAndServe(addr, countMiddleware(http.DefaultServeMux))
}

func countMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&requestCount, 1)
		next.ServeHTTP(w, r)
	})
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodGet {
        tmpl, err := template.ParseFiles("templates/index.html")
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        tmpl.Execute(w, nil)
    }
}

// --- Эндпоинт /create ---
func createHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("templates/create.html")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		class := r.FormValue("class")
		levelStr := r.FormValue("level")

		level, err := strconv.Atoi(levelStr)
		if err != nil {
			http.Error(w, "Level must be a number", 400)
			return
		}

		char := Character{Name: name, Class: class, Level: level}
		data, _ := json.Marshal(char)
		id := fmt.Sprintf("char:%d", time.Now().UnixNano())

		err = rdb.Set(ctx, id, data, 0).Err()
		if err != nil {
			http.Error(w, "Failed to save character", 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(data)
	}
}

// --- Эндпоинт /list ---
func listHandler(w http.ResponseWriter, r *http.Request) {
    keys, err := rdb.Keys(ctx, "char:*").Result()
    if err != nil {
        http.Error(w, "Redis error", 500)
        return
    }

    var chars []Character
    for _, k := range keys {
        val, _ := rdb.Get(ctx, k).Result()
        var c Character
        json.Unmarshal([]byte(val), &c)
        chars = append(chars, c)
    }

    tmpl, err := template.ParseFiles("templates/list.html")
    if err != nil {
        http.Error(w, "Failed to load template", 500)
        return
    }

    w.Header().Set("Content-Type", "text/html")
    tmpl.Execute(w, chars)
}
//func listHandler(w http.ResponseWriter, r *http.Request) {
//	keys, err := rdb.Keys(ctx, "char:*").Result()
//	if err != nil {
//		http.Error(w, "Redis error", 500)
//		return
//	}

//	var chars []Character
//	for _, k := range keys {
//		val, _ := rdb.Get(ctx, k).Result()
//		var c Character
//		json.Unmarshal([]byte(val), &c)
//		chars = append(chars, c)
//	}
//
//	w.Header().Set("Content-Type", "application/json")
//	json.NewEncoder(w).Encode(chars)
//}

// --- Эндпоинт /metrics ---
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(startTime).Seconds()
	metrics := map[string]interface{}{
		"uptime_seconds": uptime,
		"requests_total": atomic.LoadUint64(&requestCount),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// --- Эндпоинт /health ---
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}

// --- Helper ---
func getEnv(key, defaultVal string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	return v
}

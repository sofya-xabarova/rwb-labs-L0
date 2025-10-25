package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"net/http"
	"strings"

	_ "github.com/lib/pq"
	stan "github.com/nats-io/stan.go"
)

type Delivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

type Payment struct {
	Transaction  string `json:"transaction"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int    `json:"amount"`
	PaymentDT    int64  `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int    `json:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total"`
	CustomFee    int    `json:"custom_fee"`
}

type Item struct {
	ChrtID      int    `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int    `json:"price"`
	RID         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int    `json:"sale"`
	Size        string `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NmID        int    `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int    `json:"status"`
}

type Order struct {
	OrderUID          string   `json:"order_uid"`
	TrackNumber       string   `json:"track_number"`
	Entry             string   `json:"entry"`
	Delivery          Delivery `json:"delivery"`
	Payment           Payment  `json:"payment"`
	Items             []Item   `json:"items"`
	Locale            string   `json:"locale"`
	InternalSignature string   `json:"internal_signature"`
	CustomerID        string   `json:"customer_id"`
	DeliveryService   string   `json:"delivery_service"`
	ShardKey          string   `json:"shardkey"`
	SmID              int      `json:"sm_id"`
	DateCreated       string   `json:"date_created"`
	OofShard          string   `json:"oof_shard"`
}

// ----- cache loader helper -----
func LoadCacheFromDB(c *Cache, db *sql.DB) error {
	rows, err := db.Query(`SELECT data FROM orders_json`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		var o Order
		if err := json.Unmarshal(raw, &o); err != nil {
			log.Println("❌ Bad JSON in DB:", err)
			continue
		}
		_ = c.Set(o.OrderUID, &o) // адаптация: Set принимает (key, value) - но у нас Set ожидает (key string, value any) -> если ты сделал Set как выше, поправь
		// В нашей реализации Set в cache.go — принимает (key string, value any) — но в примере выше Set defined as Set(key string, value any) error
		// Если ты хочешь использовать Set(order *Order) в основном — можно вызвать c.Set(o.OrderUID, &o)
		count++
	}
	log.Printf("🔁 Cache restored: %d orders", count)
	return nil
}

func main() {
	connStr := os.Getenv("DATABASE_URL")
	//connStr := "postgres://testuser:testpass@localhost:5432/orders_db?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("DB connection error:", err)
	}
	defer db.Close()

	log.Println("✅ Connected to PostgreSQL")

	// --- Кэш: default TTL 0 (без истечения) — выбери своё значение, например 10m ---
	defaultTTL := 0 * time.Second
	cleanupInterval := 1 * time.Minute
	cache := NewCache(defaultTTL, cleanupInterval)

	// Попытка восстановить кэш из DB
	if err := LoadCacheFromDB(cache, db); err != nil {
		log.Println("⚠️ Cache restore failed:", err)
	}

	// --- NATS ---
	clusterID := "test-cluster"
	clientID := "orders-subscriber-" + randomID()
	natsURL := os.Getenv("NATS_URL")
	//natsURL := "nats://localhost:4222"

	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(natsURL))
	if err != nil {
		log.Fatal("NATS connect error:", err)
	}
	defer sc.Close()
	log.Println("✅ Connected to NATS Streaming")

	subj := "orders"
	_, err = sc.Subscribe(subj, func(m *stan.Msg) {
		var order Order
		if err := json.Unmarshal(m.Data, &order); err != nil {
			log.Println("❌ Invalid JSON:", err)
			return
		}

		// Сохраняем в DB: таблица orders_json (jsonb). Если такой таблицы нет — создайте.
		_, err := db.Exec(`INSERT INTO orders_json (order_uid, data)
                   VALUES ($1, $2)
                   ON CONFLICT (order_uid)
                   DO UPDATE SET data = EXCLUDED.data`,
			order.OrderUID, m.Data)
		if err != nil {
			log.Println("❌ DB insert error:", err)
			return
		}

		// Сохраняем в кэш (ключ = order.OrderUID, value = *Order)
		if err := cache.Set(order.OrderUID, &order); err != nil {
			log.Println("❌ Cache set error:", err)
			// но не отменяем запись в DB
		}

		log.Printf("✅ Order saved and cached: %s", order.OrderUID)
	}, stan.DurableName("orders-durable"))
	if err != nil {
		log.Fatal("Subscription error:", err)
	}
	log.Printf("👂 Subscribed to [%s]", subj)

	// HTTP handlers
	http.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/orders/")
		if id == "" {
			http.Error(w, "order_uid required", http.StatusBadRequest)
			return
		}
		v, ok := cache.Get(id)
		if !ok {
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}
		// value is any; cast
		order, _ := v.(*Order)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(order)
	})

	// ЕДИНЫЙ обработчик для /api/orders
	http.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// --- Получить список заказов (с фильтрами) ---
			q := r.URL.Query()
			product := strings.ToLower(q.Get("product"))
			city := strings.ToLower(q.Get("city"))
			customer := strings.ToLower(q.Get("customer"))

			cache.mu.RLock()
			var result []*Order
			for _, it := range cache.cacheMap {
				order, ok := it.value.(*Order)
				if !ok {
					continue
				}
				if product != "" {
					found := false
					for _, item := range order.Items {
						if strings.Contains(strings.ToLower(item.Name), product) {
							found = true
							break
						}
					}
					if !found {
						continue
					}
				}
				if city != "" && !strings.Contains(strings.ToLower(order.Delivery.City), city) {
					continue
				}
				if customer != "" && !strings.Contains(strings.ToLower(order.Delivery.Name), customer) {
					continue
				}
				result = append(result, order)
			}
			cache.mu.RUnlock()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)

		case http.MethodPost:
			// --- Добавить новый заказ ---
			var order Order
			if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
				http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
				return
			}
			if order.OrderUID == "" {
				http.Error(w, "order_uid required", http.StatusBadRequest)
				return
			}

			data, _ := json.Marshal(order)
			_, err := db.Exec(`INSERT INTO orders_json (order_uid, data)
                   VALUES ($1, $2)
                   ON CONFLICT (order_uid)
                   DO UPDATE SET data = EXCLUDED.data`,
				order.OrderUID, data)
			if err != nil {
				log.Println("❌ DB insert error:", err)
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}

			_ = cache.Set(order.OrderUID, &order)
			log.Printf("🆕 Order added via UI: %s", order.OrderUID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status":"ok"}`))

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// serve static UI
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// graceful shutdown for cache cleanup
	srv := &http.Server{Addr: ":8080"}

	// Run server in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("🌐 HTTP server started on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("🛑 Shutting down...")

	// graceful http shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)

	// Stop cache cleanup
	cache.StopCleanup()

	// Close NATS & DB done by defer
	wg.Wait()
	log.Println("✅ Shutdown complete")
}

// randomID helper для уникальности clientID
func randomID() string {
	return time.Now().Format("20060102T150405")
}

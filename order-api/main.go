package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hudl/fargo"
)

type Order struct {
	ID      string  `json:"id"`
	Total   float64 `json:"total"`
	Product string  `json:"product"`
}

var (
	ordersStore = make(map[string]Order)
	ordersMutex = &sync.RWMutex{}
	nextOrderID = 3
)

func init() {
	ordersStore["o101"] = Order{ID: "o101", Total: 1200.50, Product: "Laptop"}
	ordersStore["o102"] = Order{ID: "o102", Total: 25.00, Product: "Mouse"}
}

func ordersRouter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		handleGetOrders(w, r)
	case http.MethodPost:
		handleCreateOrder(w, r)
	case http.MethodPut:
		handleUpdateOrder(w, r)
	case http.MethodDelete:
		handleDeleteOrder(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetOrders(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/orders/")

	ordersMutex.RLock()
	defer ordersMutex.RUnlock()

	if id != "" {
		order, found := ordersStore[id]
		if !found {
			http.Error(w, `{"error": "Order not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(order)
	} else {
		allOrders := make([]Order, 0, len(ordersStore))
		for _, order := range ordersStore {
			allOrders = append(allOrders, order)
		}
		json.NewEncoder(w).Encode(allOrders)
	}
	log.Println("GET /orders request served by order-api")
}

func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var newOrder Order
	err := json.NewDecoder(r.Body).Decode(&newOrder)
	if err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	ordersMutex.Lock()
	defer ordersMutex.Unlock()

	newOrder.ID = fmt.Sprintf("o1%02d", nextOrderID)
	nextOrderID++
	ordersStore[newOrder.ID] = newOrder

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newOrder)
	log.Printf("POST /orders request served, created order %s\n", newOrder.ID)
}

func handleUpdateOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/orders/")
	if id == "" {
		http.Error(w, `{"error": "Order ID is required"}`, http.StatusBadRequest)
		return
	}

	ordersMutex.Lock()
	defer ordersMutex.Unlock()

	_, found := ordersStore[id]
	if !found {
		http.Error(w, `{"error": "Order not found"}`, http.StatusNotFound)
		return
	}

	var updatedOrder Order
	err := json.NewDecoder(r.Body).Decode(&updatedOrder)
	if err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	updatedOrder.ID = id
	ordersStore[id] = updatedOrder

	json.NewEncoder(w).Encode(updatedOrder)
	log.Printf("PUT /orders request served, updated order %s\n", id)
}

func handleDeleteOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/orders/")
	if id == "" {
		http.Error(w, `{"error": "Order ID is required"}`, http.StatusBadRequest)
		return
	}

	ordersMutex.Lock()
	defer ordersMutex.Unlock()

	if _, found := ordersStore[id]; !found {
		http.Error(w, `{"error": "Order not found"}`, http.StatusNotFound)
		return
	}

	delete(ordersStore, id)
	w.WriteHeader(http.StatusNoContent)
	log.Printf("DELETE /orders request served, deleted order %s\n", id)
}


func main() {
	eurekaConn := fargo.NewConn("http://localhost:8761/eureka")
	instance := &fargo.Instance{
		HostName:         "localhost",
		Port:             9092,
		App:              "ORDER-SERVICE",
		IPAddr:           "127.0.0.1",
		VipAddress:       "order-service",
		SecureVipAddress: "order-service",
		DataCenterInfo:   fargo.DataCenterInfo{Name: fargo.MyOwn},
		Status:           fargo.UP,
		HealthCheckUrl:   "http://localhost:9092/health",
	}

	err := eurekaConn.RegisterInstance(instance)
	if err != nil {
		log.Printf("Eureka registration failed: %v", err)
	} else {
		log.Println("Successfully registered with Eureka as ORDER-SERVICE")
	}

	go func() {
		for {
			err := eurekaConn.HeartBeatInstance(instance)
			if err != nil {
				log.Printf("Eureka lease renewal (heartbeat) failed: %v. Re-registering...", err)
				_ = eurekaConn.RegisterInstance(instance)
			}
			time.Sleep(30 * time.Second)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("De-registering from Eureka...")
		_ = eurekaConn.DeregisterInstance(instance)
		log.Println("Shutting down Go Order API.")
		os.Exit(0)
	}()

	http.HandleFunc("/orders/", ordersRouter)
	http.HandleFunc("/orders", ordersRouter)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	
	log.Println("Go Order API (CRUD) starting on port 9092...")
	if err := http.ListenAndServe(":9092", nil); err != nil {
		log.Fatal(err)
	}
}
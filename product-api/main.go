package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hudl/fargo"
)

type Product struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func productsHandler(w http.ResponseWriter, r *http.Request) {
	products := []Product{
		{ID: "p1", Name: "Laptop"},
		{ID: "p2", Name: "Mouse"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
	log.Println("Request to /products served by product-api")
}

func main() {
	eurekaConn := fargo.NewConn("http://localhost:8761/eureka")
	instance := &fargo.Instance{
		HostName:         "localhost",
		Port:             9091,
		App:              "PRODUCT-SERVICE",
		IPAddr:           "127.0.0.1",
		VipAddress:       "product-service",
		SecureVipAddress: "product-service",
		DataCenterInfo:   fargo.DataCenterInfo{Name: fargo.MyOwn},
		Status:           fargo.UP,
		HealthCheckUrl:   "http://localhost:9091/health",
	}

	err := eurekaConn.RegisterInstance(instance)
	if err != nil {
		log.Printf("Eureka registration failed: %v", err)
	} else {
		log.Println("Successfully registered with Eureka as PRODUCT-SERVICE")
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
		log.Println("Shutting down Go Product API.")
		os.Exit(0)
	}()

	http.HandleFunc("/products", productsHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	log.Println("Go Product API starting on port 9091...")
	if err := http.ListenAndServe(":9091", nil); err != nil {
		log.Fatal(err)
	}
}
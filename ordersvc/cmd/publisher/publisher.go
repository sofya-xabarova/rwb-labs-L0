package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/nats-io/stan.go"
)

func main() {
	clusterID := getEnv("NATS_CLUSTER_ID", "test-cluster")
	clientID := getEnv("NATS_CLIENT_ID", "publisher")
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	subject := getEnv("NATS_SUBJECT", "orders")

	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(natsURL))
	if err != nil {
		log.Fatalf("NATS connect error: %v", err)
	}
	defer sc.Close()

	data, err := ioutil.ReadFile("model.json")
	if err != nil {
		log.Fatalf("read model.json: %v", err)
	}

	err = sc.Publish(subject, data)
	if err != nil {
		log.Fatalf("publish error: %v", err)
	}
	fmt.Println("✅ Sent model.json to NATS Streaming channel:", subject)
}

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

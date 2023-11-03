package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nsqio/go-nsq"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/check", CheckServices).Methods("GET")

	http.Handle("/", r)

	port := "8080"
	fmt.Printf("Listening on port %s...\n", port)
	go nsqHealthCheck()
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func CheckServices(w http.ResponseWriter, r *http.Request) {
	healthCheckResult := make(map[string]string)
	checkService(&healthCheckResult, "PostgreSQL", "postgres:5432")
	checkService(&healthCheckResult, "NSQlookupd", "nsqlookupd:4160")
	checkService(&healthCheckResult, "NSQd", "nsqd:4150")
	checkService(&healthCheckResult, "NSQadmin", "nsqadmin:4171")

	for serviceName, status := range healthCheckResult {
		fmt.Fprintf(w, "%s: %s\n", serviceName, status)
	}
}

func checkService(result *map[string]string, serviceName, address string) {
	fmt.Printf("Checking %s...\n", serviceName)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("%s is unreachable: %v\n", serviceName, err)
		(*result)[serviceName] = "Unreachable"
		return
	}
	conn.Close()
	fmt.Printf("%s is reachable\n", serviceName)
	(*result)[serviceName] = "Reachable"
}

func init() {
	// Give some time for the services to start up
	time.Sleep(5 * time.Second)
}

func nsqHealthCheck() {
	// Create an NSQ producer to publish a test message
	producer, err := nsq.NewProducer("nsqd:4150", nsq.NewConfig())
	if err != nil {
		log.Fatalf("Failed to create NSQ producer: %v", err)
	}
	defer producer.Stop()

	// Create an NSQ consumer to consume the test message
	consumer, err := nsq.NewConsumer("test-topic", "test-channel", nsq.NewConfig())
	if err != nil {
		log.Fatalf("Failed to create NSQ consumer: %v", err)
	}

	// Configure the message handler for the consumer
	consumer.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		fmt.Printf("Received message: %s\n", message.Body)
		message.Finish()
		return nil
	}))

	// Connect the consumer to the NSQD server
	if err := consumer.ConnectToNSQD("nsqd:4150"); err != nil {
		log.Fatalf("Failed to connect to NSQD: %v", err)
	}

	// Publish a test message
	messageBody := []byte("Hello, NSQ!")
	if err := producer.Publish("test-topic", messageBody); err != nil {
		log.Fatalf("Failed to publish message: %v", err)
	}

	fmt.Println("Published and consumed a test message successfully!")

	// Add a sleep or any other checks to ensure that the health check keeps running

	// Block indefinitely to keep the health check running
	select {}
}

package main

import (
	"context"
	"log"
	"time"
	"wheres-my-pizza/internal/broker_messager"

	"golang.org/x/sync/errgroup"
)

func main() {
	conn, err := broker_messager.ConnectRabbitMQ("admin", "admin", "localhost:5672", "")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client, err := broker_messager.NewRabbitMQClient(conn)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	messageBus, err := client.Consume("customers_created", "email-service", false)
	if err != nil {
		panic(err)
	}
	// var blocking chan struct{}
	// go func() {
	// 	for message := range messageBus {
	// 		log.Println("New Message: %v", message)

	// 		if !message.Redelivered{
	// 			message.Nack(false, true)
	// 			continue
	// 		}
	// 		if err := message.Ack(false); err != nil {
	// 			log.Println("Failed to ack message")
	// 			continue
	// 		}
	// 		log.Printf("Ascknowledge message %s\n", message.MessageId)
	// 	}
	// }()

	// log.Println("Consuming, to close the program press CTRL+C")

	// <-blocking

	// set a timeout for 15 secs

	var blocking chan struct{}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	// errgroup allows us concurrent tasks
	g.SetLimit(10)

	go func() {
		for message := range messageBus {
			// Spawn a workerk
			msg := message

			g.Go(func() error {
				log.Printf("New Message: %v", msg)
				time.Sleep(10 * time.Second)
				if err := msg.Ack(false); err != nil {
					log.Println("Ack message failed")
					return err
				}
				log.Printf("Acknowledged message %s\n", message.MessageId)
				return nil
			})
		}
	}()

	log.Println("Consuming , usee CTRL+C to exit")
	<-blocking
}

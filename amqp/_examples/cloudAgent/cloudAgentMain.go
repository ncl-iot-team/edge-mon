package main

import (
	"flag"
	"fmt"
	"log"
	"time"
)

var (
	uri          = flag.String("uri", "amqp://test:test@10.53.21.54:5672//test", "AMQP URI")
	exchange     = flag.String("exchange", "test-exchange", "Durable, non-auto-deleted AMQP exchange name")
	exchangeType = flag.String("exchange-type", "direct", "Exchange type - direct|fanout|topic|x-custom")
	consumerTag  = flag.String("consumer-tag", "simple-consumer", "AMQP consumer tag (should not be blank)")
	reliable     = flag.Bool("reliable", true, "Wait for the publisher confirmation before exiting")
	lifetime     = flag.Duration("lifetime", 50*time.Second, "lifetime of process before shutdown (0s=infinite)")
)

var con_queues [] *Consumer



func main() {
	rec_que := []string{"Utility-Edge1"}
	sen_que := []string{"Notice-Edge1"}

	for _, queue_name := range rec_que{
		c, err := NewConsumer(*uri, *exchange, *exchangeType, queue_name, queue_name, *consumerTag)
		if err != nil {
			log.Fatalf("%s", err)
		}
		con_queues= append(con_queues, c)

	}

	// the following is the code to ask agents to send resource info
	var body string
	fmt.Scanf("%#X", &body)
	//body := "notice"
	for _, routingKey := range sen_que{
		if err := publish(*uri, *exchange, *exchangeType, routingKey, body, *reliable); err != nil {
			log.Fatalf("%s", err)
		}
		log.Printf("published %dB OK", routingKey)

	}

	if *lifetime > 0 {
		log.Printf("running for %s", *lifetime)
		time.Sleep(*lifetime)
	} else {
		log.Printf("running forever")
		select {}
	}

	log.Printf("shutting down")
	for _, c := range con_queues{
		if err := c.Shutdown(); err != nil {
			log.Fatalf("error during shutdown: %s", err)
		}
	}
}
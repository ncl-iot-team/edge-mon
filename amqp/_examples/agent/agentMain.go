package main

import (
	"flag"
	"log"
	"time"
)


var (
	uri          = flag.String("uri", "amqp://test:test@10.53.21.54:5672//test", "AMQP URI")
	exchange     = flag.String("exchange", "test-exchange", "Durable, non-auto-deleted AMQP exchange name")
	exchangeType = flag.String("exchange-type", "direct", "Exchange type - direct|fanout|topic|x-custom")
	queue        = flag.String("queue", "Notice-Edge1", "Ephemeral AMQP queue name")
	bindingKey   = flag.String("key", "Notice-Edge1", "AMQP binding key")
	routingKey   = flag.String("routingkey", "Utility-Edge1", "AMQP routing key")
	reliable     = flag.Bool("reliable", true, "Wait for the publisher confirmation before exiting")
	consumerTag  = flag.String("consumer-tag", "simple-consumer", "AMQP consumer tag (should not be blank)")
	lifetime     = flag.Duration("lifetime", 50*time.Second, "lifetime of process before shutdown (0s=infinite)")
)


func main() {
	go agent()
	c, err := NewConsumer(*uri, *exchange, *exchangeType, *queue, *bindingKey, *consumerTag)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if *lifetime > 0 {
		log.Printf("running for %s", *lifetime)
		time.Sleep(*lifetime)
	} else {
		log.Printf("running forever")
		select {}
	}
	if err := c.Shutdown(); err != nil {
		log.Fatalf("error during shutdown: %s", err)
	}

}
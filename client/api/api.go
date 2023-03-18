package api

import (
	"context"
	"encoding/json"
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
	"time"
)

var (
	RabbitMqUrl = os.Getenv("RABBIT_MQ_URL")
	QueueName   = os.Getenv("QUEUE_NAME")
)

type Record struct {
	ClothesId    int    `json:"clothesId"`
	ClothesName  string `json:"clothesName"`
	BrandName    string `json:"brandName"`
	Quantity     int    `json:"quantity"`
	CategoryName string `json:"categoryName"`
	SizeName     string `json:"sizeName"`
}

func TryDial(url string, tries int) (*amqp.Connection, error) {
	var lastError error
	for i := 0; i < tries; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			return conn, nil
		}
		lastError = err
		time.Sleep(5 * time.Second)
	}
	return nil, lastError
}

type DbExporterService struct {
	conn *amqp.Connection
}

func NewDbExporterService() (*DbExporterService, error) {
	conn, err := TryDial(RabbitMqUrl, 10)
	if err != nil {
		return nil, err
	}
	return &DbExporterService{
		conn: conn,
	}, nil
}

func (service *DbExporterService) Close() error {
	return service.conn.Close()
}

func (service *DbExporterService) Send(record *Record) error {
	ch, err := service.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(QueueName, false, false, false, false, nil)
	if err != nil {
		return err
	}

	ctx := context.Background()
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}
	err = ch.PublishWithContext(ctx,
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(body),
		})
	return err
}

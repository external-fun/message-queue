package api

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"io"
	"log"
	"os"
	"time"
)

var (
	dbPassword = os.Getenv("DB_PASSWORD")
	dbUser     = os.Getenv("DB_USER")
	dbHost     = os.Getenv("DB_HOST")
)

func getConnectionUrl() string {
	const url = "postgres://%s:%s@%s:5432/shop_db?sslmode=disable"

	return fmt.Sprintf(url, dbUser, dbPassword, dbHost)
}

type Record struct {
	ClothesId    int    `json:"clothesId"`
	ClothesName  string `json:"clothesName"`
	BrandName    string `json:"brandName"`
	Quantity     int    `json:"quantity"`
	CategoryName string `json:"categoryName"`
	SizeName     string `json:"sizeName"`
}

var (
	RabbitMqUrl = os.Getenv("RABBIT_MQ_URL")
	QueueName   = os.Getenv("QUEUE_NAME")
)

func TryDial(url string, tries int) *amqp.Connection {
	for i := 0; i < tries; i++ {
		conn, err := amqp.Dial(RabbitMqUrl)
		if err == nil {
			return conn
		}
		time.Sleep(5 * time.Second)
	}
	panic(fmt.Sprintf("couldn't dial %s", url))
}

type DbExporterService struct {
	db *sql.DB
}

func NewDbExporterService() *DbExporterService {
	db, err := sql.Open("postgres", getConnectionUrl())
	if err != nil {
		log.Println(err)
	}

	return &DbExporterService{
		db: db,
	}
}

func (service *DbExporterService) listenMq() {
	conn := TryDial(RabbitMqUrl, 10)
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Println(err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(QueueName, false, false, false, false, nil)
	if err != nil {
		log.Println(err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil)

	for msg := range msgs {
		log.Printf("Got %s\n", msg.Body)

		var record Record
		err = json.Unmarshal(msg.Body, &record)
		if err != nil {
			log.Println(err)
			continue
		}
		err = insertRecord(service.db, &record)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

func (service *DbExporterService) listenSocket() error {
	const certFile = "/cert/cert.pm"
	const keyFile = "/cert/key.pm"
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	l, err := tls.Listen("tcp", ":8081", config)
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
		}

		func() {
			defer conn.Close()
			var records []Record
			data, err := io.ReadAll(conn)
			if err != nil {
				log.Println(err)
				return
			}
			err = json.Unmarshal(data, &records)
			if err != nil {
				log.Println(err)
				return
			}

			log.Printf("Got %v\n", records)
			for _, record := range records {

				err := insertRecord(service.db, &record)
				if err != nil {
					log.Println(err)
					return
				}
			}
		}()
	}
}

func (service *DbExporterService) Listen() {
	go service.listenMq()
	go service.listenSocket()
}

func insertRow(db *sql.DB, table string, field string, value string) (int, error) {
	id := 0
	err := db.QueryRow(fmt.Sprintf(`INSERT INTO public."%s"(%s) VALUES ($1) ON CONFLICT DO NOTHING RETURNING id`, table, field), value).Scan(&id)
	if err == nil {
		return id, nil
	} else if err == sql.ErrNoRows {
		err := db.QueryRow(fmt.Sprintf(`SELECT id FROM public."%s" WHERE %s = $1`, table, field), value).Scan(&id)
		return id, err
	} else {
		return 0, err
	}
}

func insertBrand(db *sql.DB, record *Record) (int, error) {
	return insertRow(db, "Brand", "name", record.BrandName)
}

func insertCategory(db *sql.DB, record *Record) (int, error) {
	return insertRow(db, "Category", "name", record.CategoryName)
}

func insertSize(db *sql.DB, record *Record) (int, error) {
	return insertRow(db, "Size", "name", record.SizeName)
}

func insertRecord(db *sql.DB, record *Record) error {
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	brandId, err := insertBrand(db, record)
	if err != nil {
		return err
	}

	categoryId, err := insertCategory(db, record)
	if err != nil {
		return err
	}

	sizeId, err := insertSize(db, record)
	if err != nil {
		return err
	}

	// TODO: have duplicates?
	_, err = db.Exec(`INSERT INTO public."Clothes"(id, name, brand_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, record.ClothesId, record.ClothesName, brandId)
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO public."Record"(quantity, size_id, clothes_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, record.Quantity, sizeId, record.ClothesId)
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO public."ClothesAndCategory"(clothes_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, record.ClothesId, categoryId)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

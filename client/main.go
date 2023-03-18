package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/external-fun/message-queue-client/api"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	service, err := api.NewDbExporterService()
	if err != nil {
		panic(err)
	}
	defer service.Close()

	allHandler := func(w http.ResponseWriter, r *http.Request) {
		records := api.GetAll()
		var ss []string
		for _, record := range records {
			ss = append(ss, fmt.Sprintf("%#v", record))
		}
		resp := strings.Join(ss, "\n")
		w.Write([]byte(resp))
	}

	sendMqHandler := func(w http.ResponseWriter, r *http.Request) {
		records := api.GetAll()
		for _, record := range records {
			err := service.Send(&record)
			if err != nil {
				w.WriteHeader(200)
				return
			}
		}
	}

	cert, err := os.ReadFile("/cert/cert.pm")
	if err != nil {
		log.Fatal(err)
	}
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(cert); !ok {
		log.Fatal("couldn't appendCertsFromPEM")
	}
	config := &tls.Config{RootCAs: certPool}

	sendSocketHandler := func(w http.ResponseWriter, r *http.Request) {
		conn, err := tls.Dial("tcp", os.Getenv("SERVER_URL"), config)
		if err != nil {
			w.WriteHeader(200)
			log.Println(err)
			return
		}
		defer conn.Close()

		records := api.GetAll()
		data, err := json.Marshal(records)
		if err != nil {
			w.WriteHeader(200)
			log.Println(err)
			return
		}
		_, err = conn.Write(data)
		if err != nil {
			w.WriteHeader(200)
			log.Println(err)
			return
		}
	}

	http.HandleFunc("/all", allHandler)
	http.HandleFunc("/mq", sendMqHandler)
	http.HandleFunc("/socket", sendSocketHandler)
	http.ListenAndServe(":7070", nil)
}

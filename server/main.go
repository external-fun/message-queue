package main

import (
	"github.com/external-fun/message-queue-server/api"
	"net/http"
)

func main() {
	service := api.NewDbExporterService()
	go service.Listen()
	http.HandleFunc("/all", api.GetAll)
	http.HandleFunc("/clear", api.ClearAll)
	http.ListenAndServe(":8080", nil)
}

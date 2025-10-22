package agents

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func (k *K8sWebhook) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func (k *K8sWebhook) handler(w http.ResponseWriter, r *http.Request) {
	k.logger.Info("Requisição recebida", "remoteAddr", r.RemoteAddr, "path", r.URL.Path)
	var body []byte
	if r.Body != nil {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "could not read request body", http.StatusBadRequest)
			return
		}
		body = data
	}

	response, err := k.ApplyAdmission(body)
	if err != nil {
		http.Error(w, "could not apply admission", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
	log.Println("Response sent successfully")
}

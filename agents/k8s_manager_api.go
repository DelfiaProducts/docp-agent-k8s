package agents

import (
	"fmt"
	"net/http"
)

func (k *K8sManager) state(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	go k.handlerStateCheck()
	fmt.Fprintf(w, "OK")
}

func (k *K8sManager) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

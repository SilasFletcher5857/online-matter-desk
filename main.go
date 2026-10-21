package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	client, err := NewInfraiClient()
	if err != nil {
		log.Fatal(err)
	}
	svc := Service{api: client}
	http.HandleFunc("/matters", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var matter Matter
		if err := json.NewDecoder(r.Body).Decode(&matter); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := svc.Intake(r.Context(), matter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
	log.Println("legal presence listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

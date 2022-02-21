package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	mu         sync.Mutex
	PrintQueue PrintQueue
}

func NewServer() *Server {
	return &Server{
		PrintQueue: PrintQueue{
			Printers: []*Printer{},
			Requests: make([]*PrintRequest, 0),
		},
	}
}

func (s *Server) UpdatePrinter(jsonPrinter string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := &Printer{}
	err := json.Unmarshal([]byte(jsonPrinter), p)
	if err != nil {
		return err
	}

	for i, printer := range s.PrintQueue.Printers {
		if printer.ID == p.ID {
			s.PrintQueue.Printers[i] = p
			return nil
		}
	}
	s.PrintQueue.Printers = append(s.PrintQueue.Printers, p)
	return nil
}

func (s *Server) Run() {
	ctx := context.Background()
	wg := sync.WaitGroup{}

	// tick the print queue
	wg.Add(1)
	go func() {
		defer wg.Done()
		tick := time.Tick(100 * time.Millisecond)
		tick5 := time.Tick(5000 * time.Millisecond)
		for {
			select {
			case <-tick:
				s.mu.Lock()
				s.PrintQueue.Tick()
				s.mu.Unlock()
			case <-tick5:
				s.mu.Lock()
				s.PrintQueue.Print()
				s.mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	http.HandleFunc("/printQueue", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(s.PrintQueue)
	})

	http.HandleFunc("/requestPrint", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		s.PrintQueue.RequestPrint(newDocumentId())
	})
	http.ListenAndServe(":8080", nil)

}

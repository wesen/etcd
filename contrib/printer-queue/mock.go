package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

type MockPrinter struct {
	Printer              *Printer
	PrintProgressPercent int
}

func NewMockPrinter() *MockPrinter {
	id := newPrinterID()
	return &MockPrinter{
		Printer: &Printer{
			ID:        "P" + id,
			Name:      "P" + id,
			State:     PrinterIdle,
			Documents: []Document{},
			Prints:    []*Print{},
		},
	}
}

func (mp *MockPrinter) Tick() {
	// internal tick
	if mp.Printer.Assignment != nil {
		if mp.Printer.State == PrinterIdle {
			if !mp.Printer.Assignment.Claimed {
				log.Infow(fmt.Sprintf("%s: Claiming assignment", mp.Printer.Name),
					"printerID", mp.Printer.ID,
					"assignment", mp.Printer.Assignment.ID)
				mp.Printer.Assignment.Claimed = true
			} else {
				log.Infow(fmt.Sprintf("%s: Printing assignment", mp.Printer.Name),
					"printerID", mp.Printer.ID,
					"assignment", mp.Printer.Assignment.ID)
				mp.Printer.State = PrinterPrinting

				// check if document needs to be downloaded and from whom (!)
				mp.Printer.Prints = append(mp.Printer.Prints, &Print{
					ID:             newPrintId(),
					DocumentID:     mp.Printer.Assignment.DocumentID,
					AssignmentID:   &mp.Printer.Assignment.ID,
					PrintRequestID: &mp.Printer.Assignment.PrintRequestID,
					State:          PrintPrinting,
				})
				mp.PrintProgressPercent = 0
			}
		} else if mp.Printer.State == PrinterPrinting {
			if mp.PrintProgressPercent >= 100 {
				log.Infow("Printing finished",
					"printerID", mp.Printer.ID,
					"assignment", mp.Printer.Assignment.ID)
				var currentPrint *Print
				for _, _p := range mp.Printer.Prints {
					if _p.State == PrintPrinting {
						currentPrint = _p
						break
					}
				}
				if currentPrint != nil {
					log.Errorw("Could not find active print even though printing",
						"printerID", mp.Printer.ID)
					return
				}

				currentPrint.State = PrintFinished
				mp.Printer.State = PrinterIdle
			} else {
				mp.PrintProgressPercent += rand.Intn(10)
				log.Infow("Printing in progress",
					"printerID", mp.Printer.ID,
					"progress", mp.PrintProgressPercent)
			}

		}
	}
}

func (mp *MockPrinter) Start(ctx context.Context, s *Server) {
	go func() {
		tick := time.Tick(time.Second)
		for {
			select {
			case <-tick:
				mp.Tick()
				b, err := json.Marshal(mp.Printer)
				if err != nil {
					log.Errorw("Could not marshal printer",
						"printerID", mp.Printer.ID,
						"error", err)
					break
				}
				err = s.UpdatePrinter(string(b))
				if err != nil {
					log.Errorw("Could not update printer",
						"printerID", mp.Printer.ID,
						"error", err)
					break
				}
			case <-ctx.Done():
				return
			}

		}
	}()
}

package main

import (
	"fmt"
	"testing"
)

var (
	roundId = 0
)

func tick(pq *PrintQueue) {
	fmt.Printf("---------------- Round %d ----------------\n", roundId)
	roundId++
	pq.Tick()
	pq.Print()

}

func TestWholeRun(t *testing.T) {
	resetIds()
	roundId = 0
	p0 := &Printer{
		Name:  "P0",
		ID:    newPrinterID(),
		State: PrinterIdle,
	}
	doc := Document{
		ID: newDocumentId(),
	}
	doc2 := Document{
		ID: newDocumentId(),
	}
	pq := &PrintQueue{
		Printers: []*Printer{
			p0,
		},
	}

	tick(pq)

	// upload a job to printer
	p0.Documents = append(p0.Documents, doc)
	// create new print request
	pq.RequestPrint(doc.ID)

	tick(pq)

	// let's add a new couple of requests
	pq.RequestPrint(doc2.ID)
	pq.RequestPrint(doc2.ID)
	pq.RequestPrint(doc2.ID)

	tick(pq)

	// add a new printer
	pq.Printers = append(pq.Printers, &Printer{
		Name:  "P1",
		ID:    newPrinterID(),
		State: PrinterIdle,
	})
	tick(pq)
}

package main

import (
	"github.com/pborman/uuid"
	"go.uber.org/zap"
	"time"
)

var (
	log = zap.NewExample().Sugar()
)

type Document struct {
	ID uuid.UUID
}

type PrintRequest struct {
	ID         uuid.UUID
	DocumentID uuid.UUID
}

type PrintAssignment struct {
	ID             uuid.UUID
	PrintRequestID uuid.UUID
	DocumentID     uuid.UUID
	PrinterID      uuid.UUID
}

type PrintState string

const (
	PrintPrinting = "PrintPrinting"
	PrintFinished = "PrintFinished"
	PrintError    = "PrintError"
)

type Print struct {
	ID         uuid.UUID
	DocumentID uuid.UUID
	State      PrintState
}

type PrinterState string

const (
	PrinterIdle     = "PrinterIdle"
	PrinterPrinting = "PrinterPrinting"
	PrinterBusy     = "PrinterBusy"
)

type Printer struct {
	ID         uuid.UUID
	Name       string
	Documents  []Document
	State      PrinterState
	Prints     []Print
	LastSeen   time.Time
	Assignment *PrintAssignment
}

func (p *Printer) GetCurrentPrint() *Print {
	for _, _print := range p.Prints {
		if _print.State == PrintPrinting {
			return &_print
		}
	}
	return nil
}

type PrintQueue struct {
	Printers []*Printer
	Requests []PrintRequest
}

func (pq *PrintQueue) GetDocuments() []Document {
	var res []Document
	var seenIds map[string]struct{} = make(map[string]struct{})
	for _, printer := range pq.Printers {
		for _, document := range printer.Documents {
			if _, ok := seenIds[document.ID.String()]; ok {
				continue
			}
			res = append(res, document)
			seenIds[document.ID.String()] = struct{}{}
		}
	}
	return res
}

func (p *Printer) GetDocument(id uuid.UUID) (*Document, bool) {
	for _, doc := range p.Documents {
		if uuid.Equal(doc.ID, id) {
			return &doc, true
		}
	}
	return nil, false
}

// there are three conditions where a new printer assignment is created
// 1. a printer goes to idle, and there is a corresponding print request
// 2. a new print request is received, and there is an idle printer
// 3. a print assignment is discarded
//
// Also, on bootstrap, we want to schedule potential print requests

// there is another option to do scheduling, which is to go over the full
// print queue state and make sure that no print request is without an assignment
// if there is an idle printer.

// let's call this reconciliation Tick()

func (pq *PrintQueue) Tick() {
	assignments := map[string]*PrintAssignment{}
	var idlePrinters []*Printer
	for _, printer := range pq.Printers {
		if printer.State == PrinterIdle {
			log.Debugw("Printer is idle", "printer", printer.Name, "state", printer.State)
			idlePrinters = append(idlePrinters, printer)
		}

		assignment := printer.Assignment
		if assignment != nil {
			assignments[assignment.PrintRequestID.String()] = assignment
		}
	}

	idleNames := getPrinterNames(idlePrinters)
	log.Debugw("idlePrinters", "idlePrinters", idleNames)

	for _, pr := range pq.Requests {
		_, isPrAssigned := assignments[pr.ID.String()]
		log.Debugw("isPrAssigned", "pr.ID", pr.ID.String(), "isPrAssigned", isPrAssigned)
		if !isPrAssigned {
			if len(idlePrinters) > 0 {
				idlePrinter := idlePrinters[0]
				log.Debugw("assigning to", "printer", idlePrinter.ID.String(), "pr.ID", pr.ID.String())
				idlePrinter.Assignment = &PrintAssignment{
					ID:             uuid.NewRandom(),
					PrintRequestID: pr.ID,
					DocumentID:     pr.DocumentID,
					PrinterID:      idlePrinter.ID,
				}
				idlePrinters = idlePrinters[1:]
			} else {
				break
			}
		}
	}
}

func getPrinterNames(printers []*Printer) []string {
	var names []string
	for _, printer := range printers {
		names = append(names, printer.Name)
	}
	return names
}

func main() {
}

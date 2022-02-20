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

type PrintRequestState string

const (
	PrintRequestPendingAssignment = "PendingAssignment"
	PrintRequestAssigned          = "PrintRequestAssigned"
	PrintRequestPendingPrint      = "PendingPrint"
	PrintRequestPrinting          = "Printing"
	PrintRequestFinished          = "Finished"
)

type PrintRequest struct {
	ID         uuid.UUID
	State      PrintRequestState
	DocumentID uuid.UUID
}

type PrintAssignment struct {
	ID             uuid.UUID
	PrintRequestID uuid.UUID
	DocumentID     uuid.UUID
	PrinterID      uuid.UUID
	Claimed        bool
}

type PrintState string

const (
	PrintPrinting = "Printing"
	PrintFinished = "Finished"
	PrintError    = "Error"
)

type Print struct {
	ID           uuid.UUID
	DocumentID   uuid.UUID
	AssignmentID *uuid.UUID
	State        PrintState
}

type PrinterState string

const (
	PrinterIdle     = "Idle"
	PrinterPrinting = "Printing"
	PrinterBusy     = "Busy"
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
	Requests []*PrintRequest
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

func (pq *PrintQueue) assignPrintRequest(pr *PrintRequest, p *Printer) (*PrintAssignment, error) {
	if pr.State != PrintRequestPendingAssignment {
		return nil, PrintRequestAlreadyAssigned{pr}
	}
	if p.Assignment != nil {
		return nil, PrinterAlreadyAssigned{p}
	}
	if p.State != PrinterIdle {
		return nil, PrinterNotIdle{p}
	}
	pr.State = PrintRequestAssigned
	p.Assignment = &PrintAssignment{
		ID:             uuid.NewRandom(),
		PrintRequestID: pr.ID,
		DocumentID:     pr.DocumentID,
		PrinterID:      p.ID,
		Claimed:        false,
	}
	log.Debugw("assigning to", "printer", p.ID.String(), "pr.ID", pr.ID.String())

	return p.Assignment, nil
}

func (pq *PrintQueue) unassignFromPrinter(pqs *internalPrinterQueueState, printer *Printer) {
	assignment := printer.Assignment
	printRequest, ok := pqs.printRequestsByID[assignment.PrintRequestID.String()]
	if !ok {
		log.Errorw("Cannot find assigned print request"+
			" for printer", "printer", printer.ID.String(),
			"assignment", assignment.ID.String(),
			"printRequest", assignment.PrintRequestID.String())
	}
	printRequest.State = PrintRequestPendingAssignment
	printer.Assignment = nil
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
	pqs := pq.getInternalPrintQueueState()

	log.Debugw("assigned print requests", "assignedPrintRequest", pqs.assignmentsByPrintRequestID)
	idleNames := getPrinterNames(pqs.idlePrinters)
	log.Debugw("idlePrinters", "idlePrinters", idleNames)

	for _, pr := range pq.Requests {
		_, isPrAssigned := pqs.assignmentsByPrintRequestID[pr.ID.String()]
		log.Debugw("isPrAssigned", "pr.ID", pr.ID.String(), "isPrAssigned", isPrAssigned)
		if !isPrAssigned {
			if len(pqs.idlePrinters) > 0 {
				idlePrinter := pqs.idlePrinters[0]
				_, err := pq.assignPrintRequest(pr, idlePrinter)
				if err != nil {
					log.Errorw("error assigning print request", "error", err)
					continue
				}
				pqs.idlePrinters = pqs.idlePrinters[1:]
			} else {
				break
			}
		} else {
			printer, ok := pqs.printersByPrintRequestID[pr.ID.String()]
			if !ok {
				log.Errorw("Assigned print request has no printer", "pr.ID", pr.ID.String())
				continue
			}
			if printer.State == PrinterBusy {
				log.Debugw("Unassigning from printer", "pr.ID", pr.ID.String(), "printer", printer.Name)
				pq.unassignFromPrinter(pqs, printer)
			}
		}
	}
}

type internalPrinterQueueState struct {
	assignmentsByPrintRequestID map[string]*PrintAssignment
	printersByPrintRequestID    map[string]*Printer
	printRequestsByID           map[string]*PrintRequest
	idlePrinters                []*Printer
}

func (pq *PrintQueue) getInternalPrintQueueState() *internalPrinterQueueState {
	assignmentsByPrintRequestID := map[string]*PrintAssignment{}
	printersByPrintRequestID := map[string]*Printer{}
	var idlePrinters []*Printer
	for _, printer := range pq.Printers {
		if printer.State == PrinterIdle {
			log.Debugw("Printer is idle", "printer", printer.Name, "state", printer.State)
			idlePrinters = append(idlePrinters, printer)
		}

		assignment := printer.Assignment
		if assignment != nil {
			_, alreadyAssigned := assignmentsByPrintRequestID[assignment.PrintRequestID.String()]
			if alreadyAssigned {
				log.Errorw("PrintRequest is already assigned",
					"assignmentID", assignment.ID.String(),
					"printerID", printer.ID.String(),
					"previousPrinterID", assignment.PrinterID.String())
			}
			printersByPrintRequestID[assignment.PrintRequestID.String()] = printer
			assignmentsByPrintRequestID[assignment.PrintRequestID.String()] = assignment
		}
	}

	var printRequestsById = map[string]*PrintRequest{}
	for _, printRequest := range pq.Requests {
		printRequestsById[printRequest.ID.String()] = printRequest
	}
	return &internalPrinterQueueState{
		assignmentsByPrintRequestID: assignmentsByPrintRequestID,
		printersByPrintRequestID:    printersByPrintRequestID,
		idlePrinters:                idlePrinters,
		printRequestsByID:           printRequestsById,
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

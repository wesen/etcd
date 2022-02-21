package main

import (
	"fmt"
	"go.uber.org/zap"
	"time"
)

var (
	log = zap.NewExample().Sugar()
)

var (
	nextDocumentId        = 0
	nextPrintAssignmentId = 0
	nextPrintRequestId    = 0
	nextPrinterId         = 0
)

func resetIds() {
	nextDocumentId = 0
	nextPrintAssignmentId = 0
	nextPrintRequestId = 0
	nextPrinterId = 0
}

func newDocumentId() string {
	res := fmt.Sprintf("D%d", nextDocumentId)
	nextDocumentId++
	return res
}
func newPrinterAssignmentId() string {
	res := fmt.Sprintf("PA%d", nextPrintAssignmentId)
	nextPrintAssignmentId++
	return res
}
func newPrintRequestID() string {
	res := fmt.Sprintf("PR%d", nextPrintRequestId)
	nextPrintRequestId++
	return res
}
func newPrinterID() string {
	res := fmt.Sprintf("P%d", nextPrinterId)
	nextPrinterId++
	return res
}

type Document struct {
	ID string
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
	ID         string
	State      PrintRequestState
	DocumentID string
}

type PrintAssignment struct {
	ID             string
	PrintRequestID string
	DocumentID     string
	PrinterID      string
	Claimed        bool
}

type PrintState string

const (
	PrintPrinting = "Printing"
	PrintFinished = "Finished"
	PrintError    = "Error"
)

type Print struct {
	ID           string
	DocumentID   string
	AssignmentID *string
	State        PrintState
}

type PrinterState string

const (
	PrinterIdle     = "Idle"
	PrinterPrinting = "Printing"
	PrinterBusy     = "Busy"
)

type Printer struct {
	ID         string
	Name       string
	Documents  []Document
	State      PrinterState
	Prints     []*Print
	LastSeen   time.Time
	Assignment *PrintAssignment
}

func (p *Printer) GetCurrentPrint() *Print {
	for _, _print := range p.Prints {
		if _print.State == PrintPrinting {
			return _print
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
	seenIds := map[string]struct{}{}
	for _, printer := range pq.Printers {
		for _, document := range printer.Documents {
			if _, ok := seenIds[document.ID]; ok {
				continue
			}
			res = append(res, document)
			seenIds[document.ID] = struct{}{}
		}
	}
	return res
}

func (p *Printer) GetDocument(id string) (*Document, bool) {
	for _, doc := range p.Documents {
		if doc.ID == id {
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
		ID:             newPrinterAssignmentId(),
		PrintRequestID: pr.ID,
		DocumentID:     pr.DocumentID,
		PrinterID:      p.ID,
		Claimed:        false,
	}
	log.Debugw("assigning to", "printer", p.ID, "pr.ID", pr.ID)

	return p.Assignment, nil
}

func (pq *PrintQueue) unassignFromPrinter(pqs *internalPrinterQueueState, printer *Printer) {
	assignment := printer.Assignment
	printRequest, ok := pqs.printRequestsByID[assignment.PrintRequestID]
	if !ok {
		log.Errorw("Cannot find assigned print request"+
			" for printer", "printer", printer.ID,
			"assignment", assignment.ID,
			"printRequest", assignment.PrintRequestID)
	} else {
		printRequest.State = PrintRequestPendingAssignment
	}
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
	pqs := pq.computeInternalPrintQueueState()

	log.Debugw("assigned print requests", "assignedPrintRequest", pqs.assignmentsByPrintRequestID)
	idleNames := getPrinterNames(pqs.idlePrinters)
	log.Debugw("idlePrinters", "idlePrinters", idleNames)

	for _, pr := range pq.Requests {
		_, isPrAssigned := pqs.assignmentsByPrintRequestID[pr.ID]
		log.Debugw("isPrAssigned", "pr.ID", pr.ID, "isPrAssigned", isPrAssigned)
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
			printer, ok := pqs.printersByPrintRequestID[pr.ID]
			if !ok {
				log.Errorw("Assigned print request has no printer", "pr.ID", pr.ID)
				continue
			}
			if printer.State == PrinterBusy {
				log.Debugw("Unassigning from printer", "pr.ID", pr.ID, "printer", printer.Name)
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
	currentPrintsByPrinterId    map[string]*Print
	assignedPrintByPrinterId    map[string]*Print
}

func (pq *PrintQueue) computeInternalPrintQueueState() *internalPrinterQueueState {
	pqs := &internalPrinterQueueState{
		assignmentsByPrintRequestID: make(map[string]*PrintAssignment),
		printersByPrintRequestID:    make(map[string]*Printer),
		printRequestsByID:           make(map[string]*PrintRequest),
		currentPrintsByPrinterId:    make(map[string]*Print),
		assignedPrintByPrinterId:    make(map[string]*Print),
	}

	for _, printer := range pq.Printers {
		if printer.State == PrinterIdle {
			pqs.idlePrinters = append(pqs.idlePrinters, printer)
		}

		assignment := printer.Assignment
		if assignment != nil {
			_, alreadyAssigned := pqs.assignmentsByPrintRequestID[assignment.PrintRequestID]
			if alreadyAssigned {
				log.Errorw("PrintRequest is already assigned",
					"assignmentID", assignment.ID,
					"printerID", printer.ID,
					"previousPrinterID", assignment.PrinterID)
			}
			pqs.printersByPrintRequestID[assignment.PrintRequestID] = printer
			pqs.assignmentsByPrintRequestID[assignment.PrintRequestID] = assignment
		}

		for _, _print := range printer.Prints {
			if _print.State == PrintPrinting {
				_, alreadyPrinting := pqs.currentPrintsByPrinterId[printer.ID]
				if alreadyPrinting {
					log.Errorw("Printer is already printing",
						"printerID", printer.ID,
						"previousPrintID", _print.ID)
				}
				pqs.currentPrintsByPrinterId[printer.ID] = _print
			}
			if assignment != nil && _print.AssignmentID != nil && *_print.AssignmentID == assignment.ID {
				_, alreadyAssigned := pqs.assignedPrintByPrinterId[printer.ID]
				if alreadyAssigned {
					log.Errorw("Printer is already assigned",
						"printerID", printer.ID,
						"previousPrintID", _print.ID)
				}
				pqs.assignedPrintByPrinterId[printer.ID] = _print
			}
		}
	}

	for _, printRequest := range pq.Requests {
		pqs.printRequestsByID[printRequest.ID] = printRequest
	}
	return pqs
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

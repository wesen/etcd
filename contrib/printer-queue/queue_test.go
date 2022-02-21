package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func createDocIds(count int) []string {
	var ids []string
	for i := 0; i < count; i++ {
		ids = append(ids, newDocumentId())
	}
	return ids
}

var (
	docIds        = createDocIds(10)
	printRequests = []*PrintRequest{
		{
			ID:         newPrintRequestID(),
			DocumentID: docIds[0],
			State:      PrintRequestPendingAssignment,
		},
		{
			ID:         newPrintRequestID(),
			DocumentID: docIds[1],
			State:      PrintRequestPendingAssignment,
		},
		{
			ID:         newPrintRequestID(),
			DocumentID: docIds[2],
			State:      PrintRequestPendingAssignment,
		},
		{
			ID:         newPrintRequestID(),
			DocumentID: docIds[3],
			State:      PrintRequestPendingAssignment,
		}}
)

func createPrinterQueue() PrintQueue {
	resetIds()
	pq := PrintQueue{
		Printers: []*Printer{
			{
				ID:    newPrinterID(),
				Name:  "P0",
				State: PrinterIdle,
				Documents: []Document{
					{ID: docIds[0]},
					{ID: docIds[1]},
				},
			},
			{
				ID:    newPrinterID(),
				Name:  "P1",
				State: PrinterIdle,
				Documents: []Document{
					{ID: docIds[2]},
					{ID: docIds[3]},
				},
			},
		},
	}
	return pq
}
func createMultiPrinterQueue(numPrinters int) PrintQueue {
	resetIds()
	printers := make([]*Printer, numPrinters)
	for i := range printers {
		printers[i] = &Printer{
			ID:    newPrinterID(),
			Name:  fmt.Sprintf("P%d", i),
			State: PrinterIdle,
			Documents: []Document{
				{ID: docIds[i]},
			},
		}
	}
	return PrintQueue{
		Printers: printers,
	}
}
func createPrintAssignment(i int) *PrintAssignment {
	return &PrintAssignment{
		ID:             newPrinterAssignmentId(),
		PrintRequestID: fmt.Sprintf("PR%d", i),
		PrinterID:      fmt.Sprintf("P%d", i),
		DocumentID:     fmt.Sprintf("D%d", i),
	}
}

// Test that we correctly merge the document lists of 2 printers
func TestListDocumentsSimple(t *testing.T) {
	pq := createPrinterQueue()

	allDocs := pq.GetDocuments()
	allDocIds := getDocIds(allDocs)

	assert.ElementsMatch(t, allDocIds, docIds[:4])
}

// Test that we correctly remove duplicates from the document list
func TestListDocumentsDouble(t *testing.T) {
	pq := createPrinterQueue()
	pq.Printers[0].Documents[0].ID = docIds[2]

	allDocs := pq.GetDocuments()
	allDocIds := getDocIds(allDocs)

	assert.ElementsMatch(t, allDocIds, docIds[1:4])
}

// Test that an idle printer gets assigned a print request without assignment
func TestPrintAssignmentWhenPrinterIdle(t *testing.T) {
	pq := createPrinterQueue()
	pq.Requests = printRequests[:1]
	pq.Printers[0].State = PrinterIdle
	pq.Printers[1].State = PrinterPrinting

	pq.Tick()

	assignment := pq.Printers[0].Assignment
	assert.NotNil(t, assignment)
	assert.EqualValues(t, printRequests[0].ID, assignment.PrintRequestID)
	assert.EqualValues(t, pq.Printers[0].ID,
		assignment.PrinterID)
	assert.EqualValues(t, docIds[0], assignment.DocumentID)

	assert.EqualValues(t, pq.Requests[0].State, PrintRequestAssigned)
}

// Test that a printer that is doing anything other than being idle or printing
// their assignment gets their assignment removed
func TestPrinterBusyAssignmentRemoved(t *testing.T) {
	pq := createPrinterQueue()
	pq.Requests = printRequests[:3]
	pq.Printers[0].State = PrinterBusy
	pq.Printers[0].Assignment = createPrintAssignment(0)
	pq.Requests[0].State = PrintRequestAssigned

	pq.Tick()

	assert.Nil(t, pq.Printers[0].Assignment)
	assert.EqualValues(t, pq.Requests[0].State, PrintRequestPendingAssignment)
}

// Test that a printer that is printing another document than the requested
// print gets their assignment removed
func TestPrinterPrintingOtherDocumentAssignmentRemoved(t *testing.T) {
	pq := createPrinterQueue()
	pq.Requests = printRequests[:3]
	pq.Printers[0].State = PrinterPrinting
	pq.Printers[0].Prints = []*Print{
		{
			DocumentID: docIds[1],
			State:      PrintPrinting,
		},
	}
	pq.Printers[0].Assignment = createPrintAssignment(0)
	pq.Requests[0].State = PrintRequestAssigned

	pq.Tick()

	assert.Nil(t, pq.Printers[0].Assignment)
	assert.EqualValues(t, pq.Requests[0].State, PrintRequestPendingAssignment)
}

// Test that a printer has no current print, and has printed other prints
// in the past, the assignment does not get removed
func TestPrinterPrintedOtherDocumentAssignmentNotRemoved(t *testing.T) {
	pq := createPrinterQueue()
	pq.Requests = printRequests[:3]
	pq.Printers[0].State = PrinterPrinting
	pq.Printers[0].Prints = []*Print{
		{
			DocumentID: docIds[0],
			State:      PrintFinished,
		},
		{
			DocumentID: docIds[1],
			State:      PrintFinished,
		},
	}
	pq.Printers[0].Assignment = createPrintAssignment(0)
	pq.Requests[0].State = PrintRequestAssigned

	pq.Tick()

	assert.NotNil(t, pq.Printers[0].Assignment)
	assert.EqualValues(t, pq.Requests[0].State, PrintRequestAssigned)
}

// Test that a finished print removes the corresponding assignment
// and print request
func TestPrintFinishedErroredUpdatePrintRequest(t *testing.T) {
	pq := createPrinterQueue()
	pq.Requests = printRequests[:2]

	pq.Printers[0].State = PrinterIdle
	pq.Printers[0].Assignment = createPrintAssignment(0)
	assignment := pq.Printers[0].Assignment
	pq.Requests[0].State = PrintRequestPrinting

	pq.Printers[1].State = PrinterIdle
	pq.Printers[1].Assignment = createPrintAssignment(1)
	assignment2 := pq.Printers[1].Assignment
	pq.Requests[1].State = PrintRequestPrinting

	pq.Printers[0].Prints = []*Print{
		{
			ID:             newPrintId(),
			AssignmentID:   &assignment.ID,
			PrintRequestID: &assignment.PrintRequestID,
			State:          PrintFinished,
			DocumentID:     assignment.DocumentID,
		},
	}
	pq.Printers[1].Prints = []*Print{
		{
			ID:             newPrintId(),
			AssignmentID:   &assignment2.ID,
			PrintRequestID: &assignment2.PrintRequestID,
			State:          PrintError,
			DocumentID:     assignment2.DocumentID,
		},
	}

	pq.Tick()
	assert.Nil(t, pq.Printers[0].Assignment)
	assert.Nil(t, pq.Printers[1].Assignment)
	assert.EqualValues(t, pq.Requests[0].State, PrintRequestFinished)
	assert.EqualValues(t, pq.Requests[1].State, PrintRequestFinished)
}

func TestOnlyAssignPendingAssignmentPrintRequests(t *testing.T) {
	pq := createPrinterQueue()
	pq.Requests = printRequests
	pq.Requests[0].State = PrintRequestAssigned
	pq.Requests[1].State = PrintRequestAssigned
	pq.Requests[2].State = PrintRequestPendingPrint
	pq.Requests[3].State = PrintRequestPrinting
	pq.Printers[0].State = PrinterIdle

	pq.Tick()

	assignment := pq.Printers[0].Assignment
	assert.Nil(t, assignment)
}

func TestDontAssignToIdleAssignedPrinter(t *testing.T) {
	pq := createPrinterQueue()

	p0 := pq.Printers[0]
	p1 := pq.Printers[1]
	pq.Requests = printRequests
	p0.State = PrinterIdle

	pq.Tick()
	assert.NotNil(t, p0.Assignment)
	assert.EqualValues(t, p0.Assignment.PrintRequestID, pq.Requests[0].ID)
	assert.EqualValues(t, p1.Assignment.PrintRequestID, pq.Requests[1].ID)

	assert.EqualValues(t, PrintRequestAssigned, pq.Requests[0].State)
	assert.EqualValues(t, PrintRequestAssigned, pq.Requests[1].State)
	assert.EqualValues(t, PrintRequestPendingAssignment, pq.Requests[2].State)
	assert.EqualValues(t, PrintRequestPendingAssignment, pq.Requests[3].State)
}

// XXX later, error handling / inconsistent printer states
// Test that a printer printing their assignment without claiming it are reported
// as an error

package main

import (
	"fmt"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	startId            = 1
	printerIds         = createIds(3)
	docIds             = createIds(10)
	printRequestIds    = createIds(4)
	printAssignmentIds = createIds(4)

	printRequests = []*PrintRequest{
		{
			ID:         printRequestIds[0],
			DocumentID: docIds[0],
			State:      PrintRequestPendingAssignment,
		},
		{
			ID:         printRequestIds[1],
			DocumentID: docIds[1],
			State:      PrintRequestPendingAssignment,
		},
		{
			ID:         printRequestIds[2],
			DocumentID: docIds[2],
			State:      PrintRequestPendingAssignment,
		},
		{
			ID:         printRequestIds[3],
			DocumentID: docIds[3],
			State:      PrintRequestPendingAssignment,
		}}
)

func createIds(len int) []uuid.UUID {
	ids := make([]uuid.UUID, len)
	for i := range ids {
		ids[i] = uuid.Parse(fmt.Sprintf("00000000-0000-0000-0000-00000000%04d", startId))
		startId++
	}
	return ids
}
func createPrinterQueue() PrintQueue {
	pq := PrintQueue{
		Printers: []*Printer{
			{
				ID:    printerIds[0],
				Name:  "P0",
				State: PrinterIdle,
				Documents: []Document{
					{ID: docIds[0]},
					{ID: docIds[1]},
				},
			},
			{
				ID:    printerIds[1],
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
func uuidsToStrings(uuids []uuid.UUID) []string {
	strs := make([]string, len(uuids))
	for i := range uuids {
		strs[i] = uuids[i].String()
	}
	return strs
}
func createPrintAssignment(i int, docIds []uuid.UUID) *PrintAssignment {
	return &PrintAssignment{
		ID:             printAssignmentIds[i],
		PrintRequestID: printRequestIds[i],
		PrinterID:      printerIds[i],
		DocumentID:     docIds[i],
	}
}
func getDocIds(docs []Document) []string {
	var ids []string
	for _, v := range docs {
		ids = append(ids, v.ID.String())
	}
	return ids
}

// Test that we correctly merge the document lists of 2 printers
func TestListDocumentsSimple(t *testing.T) {
	pq := createPrinterQueue()

	allDocs := pq.GetDocuments()
	allDocIds := getDocIds(allDocs)

	assert.ElementsMatch(t, allDocIds, uuidsToStrings(docIds[:4]))
}

// Test that we correctly remove duplicates from the document list
func TestListDocumentsDouble(t *testing.T) {
	pq := createPrinterQueue()
	pq.Printers[0].Documents[0].ID = docIds[2]

	allDocs := pq.GetDocuments()
	allDocIds := getDocIds(allDocs)

	assert.ElementsMatch(t, allDocIds, uuidsToStrings(docIds[1:4]))
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
	assert.EqualValues(t, printRequestIds[0].String(),
		assignment.PrintRequestID.String())
	assert.EqualValues(t, printerIds[0].String(),
		assignment.PrinterID.String())
	assert.EqualValues(t, docIds[0].String(), assignment.DocumentID.String())

	assert.EqualValues(t, pq.Requests[0].State, PrintRequestAssigned)
}

// Test that a printer that is doing anything other than being idle or printing
// their assignment gets their assignment removed
func TestPrinterBusyAssignmentRemoved(t *testing.T) {
	pq := createPrinterQueue()
	pq.Requests = printRequests[:3]
	pq.Printers[0].State = PrinterBusy
	pq.Printers[0].Assignment = createPrintAssignment(0, docIds)
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
	pq.Printers[0].Prints = []Print{
		{
			DocumentID: docIds[1],
			State:      PrintPrinting,
		},
	}
	pq.Printers[0].Assignment = createPrintAssignment(0, docIds)
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
	pq.Printers[0].Prints = []Print{
		{
			DocumentID: docIds[0],
			State:      PrintFinished,
		},
		{
			DocumentID: docIds[1],
			State:      PrintFinished,
		},
	}
	pq.Printers[0].Assignment = createPrintAssignment(0, docIds)
	pq.Requests[0].State = PrintRequestAssigned

	pq.Tick()

	assert.NotNil(t, pq.Printers[0].Assignment)
	assert.EqualValues(t, pq.Requests[0].State, PrintRequestAssigned)
}

// Test that a finished print removes the corresponding assignment
// and print request

// XXX later, error handling / inconsistent printer states
// Test that a printer printing their assignment without claiming it are reported
// as an error

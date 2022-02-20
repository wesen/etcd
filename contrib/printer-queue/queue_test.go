package main

import (
	"fmt"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	startId         = 1
	printerIds      = createIds(3)
	docIds          = createIds(10)
	printRequestIds = createIds(4)
)

func createIds(len int) []uuid.UUID {
	ids := make([]uuid.UUID, len)
	for i := range ids {
		ids[i] = uuid.Parse(fmt.Sprintf("00000000-0000-0000-0000-00000000%04d", startId))
		startId++
	}
	return ids
}

func createPrinterQueue() ([]uuid.UUID, PrintQueue) {
	pq := PrintQueue{
		Printers: []*Printer{
			{
				ID:   printerIds[0],
				Name: "P0",
				Documents: []Document{
					{ID: docIds[0]},
					{ID: docIds[1]},
				},
			},
			{
				ID:   printerIds[1],
				Name: "P1",
				Documents: []Document{
					{ID: docIds[2]},
					{ID: docIds[3]},
				},
			},
		},
	}
	return docIds, pq
}

func uuidsToStrings(uuids []uuid.UUID) []string {
	strs := make([]string, len(uuids))
	for i := range uuids {
		strs[i] = uuids[i].String()
	}
	return strs
}

func getDocIds(docs []Document) []string {
	var ids []string
	for _, v := range docs {
		ids = append(ids, v.ID.String())
	}
	return ids
}

func TestListDocumentsSimple(t *testing.T) {
	docIds, pq := createPrinterQueue()

	allDocs := pq.GetDocuments()
	allDocIds := getDocIds(allDocs)

	assert.ElementsMatch(t, allDocIds, uuidsToStrings(docIds[:4]))
}

func TestListDocumentsDouble(t *testing.T) {
	docIds, pq := createPrinterQueue()
	pq.Printers[0].Documents[0].ID = docIds[2]

	allDocs := pq.GetDocuments()
	allDocIds := getDocIds(allDocs)

	assert.ElementsMatch(t, allDocIds, uuidsToStrings(docIds[1:4]))
}

func TestPrintAssignmentWhenPrinterIdle(t *testing.T) {
	docIds, pq := createPrinterQueue()
	pq.Requests = []PrintRequest{{
		ID:         printRequestIds[0],
		DocumentID: docIds[0],
	}}
	pq.Printers[0].State = PrinterIdle
	pq.Printers[1].State = PrinterPrinting

	log.Debugw("printers", "printers", pq.Printers)

	pq.Tick()

	assignment := pq.Printers[0].Assignment
	assert.NotNil(t, assignment)
	assert.Equal(t, printRequestIds[0].String(),
		assignment.PrintRequestID.String())
	assert.Equal(t, printerIds[0].String(),
		assignment.PrinterID.String())
	assert.Equal(t, docIds[0].String(), assignment.DocumentID.String())
}

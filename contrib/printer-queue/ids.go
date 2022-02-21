package main

import "fmt"

var (
	nextDocumentId        = 0
	nextPrintAssignmentId = 0
	nextPrintRequestId    = 0
	nextPrinterId         = 0
	nextPrintId           = 0
)

func resetIds() {
	nextDocumentId = 0
	nextPrintAssignmentId = 0
	nextPrintRequestId = 0
	nextPrinterId = 0
	nextPrintId = 0
}

func newDocumentId() string {
	res := fmt.Sprintf("D%d", nextDocumentId)
	nextDocumentId++
	return res
}
func newPrintId() string {
	res := fmt.Sprintf("p%d", nextPrintId)
	nextPrintId++
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

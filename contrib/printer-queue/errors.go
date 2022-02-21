package main

import "fmt"

type PrintRequestAlreadyAssigned struct {
	PrintRequest *PrintRequest
}

func (praa PrintRequestAlreadyAssigned) Error() string {
	return fmt.Sprintf("Print request already assigned, state: %s", praa.PrintRequest.State)
}

type PrinterAlreadyAssigned struct {
	Printer *Printer
}

func (paa PrinterAlreadyAssigned) Error() string {
	assignment := paa.Printer.Assignment
	return fmt.Sprintf("Printer already assigned %s (PR: %s)", assignment.ID, assignment.PrintRequestID)
}

type PrinterNotIdle struct {
	Printer *Printer
}

func (pni PrinterNotIdle) Error() string {
	return fmt.Sprintf("Printer is not idle: %s", pni.Printer.State)
}

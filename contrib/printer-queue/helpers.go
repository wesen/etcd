package main

import (
	"fmt"
	"strings"
)

func getDocIds(docs []Document) []string {
	var ids []string
	for _, v := range docs {
		ids = append(ids, v.ID)
	}
	return ids
}

func (pq *PrintQueue) Print() {
	fmt.Printf("Printers:\n")
	for _, printer := range pq.Printers {
		printer.Print("  ")
	}
	fmt.Printf("Requests:\n")
	for _, pr := range pq.Requests {
		pr.Print("  ")
	}
	fmt.Printf("------\n")
}

func (pr *PrintRequest) Print(indent string) {
	fmt.Printf("%sPrint Request: %s\n", indent, pr.ID)
	fmt.Printf("%s  State: %s\n", indent, pr.State)
	fmt.Printf("%s  DocumentID: %v\n", indent, pr.DocumentID)
}

func (a *PrintAssignment) Print(indent string) {
	fmt.Printf("%sAssignment: %s\n", indent, a.ID)
	fmt.Printf("%s  PrinterID: %s\n", indent, a.PrinterID)
	fmt.Printf("%s  PrintRequestID: %s\n", indent, a.PrintRequestID)
	fmt.Printf("%s  Claimed: %v\n", indent, a.Claimed)
	fmt.Printf("%s  DocumentID: %s\n", indent, a.DocumentID)
}

func (p *Print) Print(indent string) {
	fmt.Printf("%sPrint: %s\n", indent, p.ID)
	fmt.Printf("%s  State: %s\n", indent, p.State)
	if p.PrintRequestID != nil {
		fmt.Printf("%s  PrintRequestID: %s\n", indent, *p.PrintRequestID)
	} else {
		fmt.Printf("%s  \n", indent)
	}
	fmt.Printf("%s  DocumentID: %v\n", indent, p.DocumentID)
	if p.AssignmentID == nil {
		fmt.Printf("%s  No assignment\n", indent)
	} else {
		fmt.Printf("%s  AssignmentID: %s\n", indent, *p.AssignmentID)
	}
}

func (p *Printer) Print(indent string) {
	fmt.Printf("%sPrinter: %s (%s)\n", indent, p.ID, p.Name)
	fmt.Printf("%s  State: %s\n", indent, p.State)
	fmt.Printf("%s  LastSeen: %s\n", indent, p.LastSeen.Format("2006-01-02T15:04:05-0700"))
	if p.Assignment != nil {
		fmt.Printf("%s  Assignment:\n", indent)
		p.Assignment.Print(indent + "  ")
	} else {
		fmt.Printf("%s  No assignment\n", indent)
	}
	var currentPrint *Print
	for _, _print := range p.Prints {
		if _print.State == PrintPrinting {
			currentPrint = _print
			break
		}
	}
	if currentPrint != nil {
		fmt.Printf("%s  Current print:\n", indent)
		currentPrint.Print(indent + "  ")
	} else {
		fmt.Printf("%s  No current print\n", indent)
	}
	fmt.Printf("%s  Documents: %s\n", indent, strings.Join(getDocIds(p.Documents), ", "))
}

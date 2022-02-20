# Reformulating for 2D printers
## Overview
We want to design a distributed print queue to print PDFs across a fleet of network connected printers.

## Desired behaviour
### Glossary

* *printer*: a machine that can:
    * store *documents*
    * print *print assignments*
    * upload/download *documents* to another *printer*
* *documents*: file + metadata that have been uploaded to one or many *printers*
    * a *document* is: (documentGuid, name, pageCount) + file
* *print request*: request to print a certain *document*
    * a *print request* is (printRequestGuid, documentGuid)
* *print assignment*: requesting a *printer* to print a certain *print requests*, aka a certain document
    * a *print assignment* is (guid, printerGuid, printRequestGuid)
* a printer actually putting ink to paper is said to execute a *print*
* *Printers* on the same network function as a unified print queue
    * Printing to any *printer* in the fleet will schedule that *document* to be printed across the fleet.

### Behaviour specification

* *Printers* can be in the following states:
    * *IDLE, PRINTING, BUSY* (otherwise doing something that prevents printing, say cleaning its print heads)
    * The state also includes the set of *documents* available on the *printer* itself, which is a list of [*documentGuid*]
    * The state also includes the printGuid of the printing *print* (and potentially some metadata about the *print* itself).
* *Printers* signal availability using *heartbeats*
* *Print requests* get assigned to *printers* according to the following rule:
    * A new *print request* is created, and there is a *IDLE printer* with matching paper feed (enough pages, etc…). There is some heuristic for which printers to prefer.
    * A *printer* becomes *IDLE*
    * An existing *print assignment* is retired
* A *print assignment* has a lifetime after which it gets discarded if not claimed
* *Print assignment* get discarded when:
    * They haven’t started printing in the given timeout
    * The *printer* they are assigned to disappears
    * The *printer* they are assigned to leaves *IDLE* state and doesn’t enter the *PRINTING* state executing a *print* of the requested *documentId*
* There is at most one *print assignment* per *printer*, and at most one *print assignment* per *print request*
* *Printers* copy the *document* needed to *print* their *print request* from the *printers* who have said *document*

## Implementation sketch
I want to use distributed consensus on top of Raft to implement the business logic to both synchronize the print queue across the fleet, and schedule print assignments.

### Print queue state

We replicate the following print queue state. This is the state that will be snapshot to disk.

* *printers*
    * printerGuid
    * state: *IDLE | PRINTING | BUSY*
    * *prints*, list of:
        * printGuid
        * documentGuid
        * state: *PRINTING | FINISHED | ERROR | ABORTED*
        * progressInformation
    * *documents*: list of
        * documentGuid
    * *print assignment*, optional:
        * printRequestId
        * state: WAITING_FOR_CLAIM, CLAIMED
    * paperFeed information
    * lastSeen
* *print requests*, list of
    * (jobGuid, createdBy, createdAt)

### Log entries

* UpdatePrinterState
* CreatePrintRequest
* DeletePrintRequest
* CreatePrintAssignment
* ClaimPrintAssignment
* ReassignPrintAssignment
* DeletePrintAssignment

### API
* `GET /state` returns the entire snapshot state
* `GET /printers/$i/assignment`
    * can be watched by printers to see if a print gets assigned to them
* `POST /printers/$id/heartbeat`
    * Update printer state
* `POST /printRequests/ (jobGuid)`
    * Allows a user to request the printing of a job
* `DELETE /printerRequest/XXX`
* `POST /printers/$id/assignment`, used by printers to claim an assignment



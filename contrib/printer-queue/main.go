package main

import (
	"context"
	"encoding/json"
	"fmt"
)

func main() {
	s := NewServer()
	mp1 := NewMockPrinter()
	b, err := json.Marshal(mp1.Printer)
	if err != nil {
		panic(err)
	}
	fmt.Printf("json: %s\n", string(b))
	mp1.Start(context.TODO(), s)
	s.Run()
}

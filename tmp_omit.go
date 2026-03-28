package main

import (
	"encoding/json"
	"fmt"
)

type E struct {
	Type      string `json:"type"`
	Committed *bool  `json:"committed,omitempty"`
}

func main() {
	f, t := false, true
	for _, v := range []*bool{nil, &f, &t} {
		b, _ := json.Marshal(E{Type: "x", Committed: v})
		fmt.Println(string(b))
	}
}

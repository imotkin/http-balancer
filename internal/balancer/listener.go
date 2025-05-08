package balancer

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func NewListener(port int, w io.Writer) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(
			w,
			"Listener (:%d) got a %s request (%d bytes)",
			port, r.Method, r.ContentLength,
		)
	})

	log.Println("Started a listener on port:", port)

	go http.ListenAndServe(
		fmt.Sprintf("localhost:%d", port), handler,
	)
}

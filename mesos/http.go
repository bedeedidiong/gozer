package mesos

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func startServing(d *Driver) {

	// TODO(weingart): Grab an ephemeral port for this instead and toss it into MesosMaster
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprint(rw, "OK\r\n")
	})
	mux.Handle("/", d)

	d.config.Log.Info.Println("Listening on port", d.pidPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", d.pidPort), mux); err != nil {
		d.config.Log.Error.Fatal("failed to start listening on port", d.pidPort)
	}
}

func (d *Driver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Add("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		d.config.Log.Error.Println("received request with unexpected method. want \"POST\", got", r.Method)
		return
	}

	pathElements := strings.Split(r.URL.Path, "/")

	if pathElements[1] != d.config.FrameworkName {
		w.WriteHeader(http.StatusNotFound)
		errStr := fmt.Sprintf("unexpected path. want %q, got %q", d.config.FrameworkName, pathElements[1])
		d.config.Log.Error.Println(errStr)
		w.Write([]byte(errStr))
		return
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		d.config.Log.Error.Printf("failed to read body from request %+v: %+v", r, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	event, err := bytesToEvent(pathElements[2], body)
	if err != nil {
		d.config.Log.Error.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	d.events <- event

	w.WriteHeader(http.StatusOK)
}

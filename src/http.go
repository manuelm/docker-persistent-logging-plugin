package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/docker/docker/daemon/logger"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/go-plugins-helpers/sdk"
)


type StartLoggingRequest struct {
	File string
	Info logger.Info
}

type StopLoggingRequest struct {
	File string
}

type ReadLogsRequest struct {
	Info   logger.Info
	Config logger.ReadConfig
}

type Response struct {
	Err string
}

type CapabilitiesResponse struct {
	ReadLogs bool
}


func respond(err error, writer http.ResponseWriter) {
	var response Response
	if err != nil {
		response.Err = err.Error()
	}
	json.NewEncoder(writer).Encode(&response)
}


// handlers implements the LogDriver protocol with some sanity checking. The protocol is detailed here:
// https://docs.docker.com/engine/extend/plugins_logging/
func handlers(handler *sdk.Handler, driver *driver) {

	handler.HandleFunc("/LogDriver.StartLogging", func(writer http.ResponseWriter, request *http.Request) {
		var req StartLoggingRequest
		if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Info.ContainerImageName == "" {
			respond(errors.New("must provide container image name in log context"), writer)
			return
		}

		err := driver.StartLogging(req.File, req.Info)
		respond(err, writer)
	})

	handler.HandleFunc("/LogDriver.StopLogging", func(writer http.ResponseWriter, request *http.Request) {
		var req StopLoggingRequest
		if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		err := driver.StopLogging(req.File)
		respond(err, writer)
	})

	handler.HandleFunc("/LogDriver.Capabilities", func(writer http.ResponseWriter, request *http.Request) {
	    var response = CapabilitiesResponse{ReadLogs: true}
		json.NewEncoder(writer).Encode(&response)
	})

	handler.HandleFunc("/LogDriver.ReadLogs", func(writer http.ResponseWriter, response *http.Request) {
		var req ReadLogsRequest
		if err := json.NewDecoder(response.Body).Decode(&req); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		stream, err := driver.ReadLogs(req.Info, req.Config)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		defer stream.Close()

		writer.Header().Set("Content-Type", "application/x-json-stream")
		wf := ioutils.NewWriteFlusher(writer)
		io.Copy(wf, stream)
	})
}

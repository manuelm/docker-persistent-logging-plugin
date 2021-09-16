package main

import (
    "encoding/json"
    "io"
    "net/http"

    "github.com/docker/docker/daemon/logger"
    "github.com/docker/docker/pkg/ioutils"
    "github.com/docker/go-plugins-helpers/sdk"
)


type startLoggingRequest struct {
    File string
    Info logger.Info
}

type stopLoggingRequest struct {
    File string
}

type readLogsRequest struct {
    Info   logger.Info
    Config logger.ReadConfig
}

type response struct {
    Err string
}

type capabilitiesResponse struct {
    Err string
    Cap logger.Capability
}


func respond(err error, writer http.ResponseWriter) {
    var resp response
    if err != nil {
        resp.Err = err.Error()
    }
    json.NewEncoder(writer).Encode(&resp)
}


// handlers implements the LogDriver protocol with some basic error handling. The protocol is detailed here:
// https://docs.docker.com/engine/extend/plugins_logging/
func handlers(handler *sdk.Handler, driver *driver) {

    handler.HandleFunc("/LogDriver.StartLogging", func(writer http.ResponseWriter, request *http.Request) {
        var req startLoggingRequest
        if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
            http.Error(writer, err.Error(), http.StatusBadRequest)
            return
        }
        err := driver.StartLogging(req.File, req.Info)
        respond(err, writer)
    })

    handler.HandleFunc("/LogDriver.StopLogging", func(writer http.ResponseWriter, request *http.Request) {
        var req stopLoggingRequest
        if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
            http.Error(writer, err.Error(), http.StatusBadRequest)
            return
        }
        err := driver.StopLogging(req.File)
        respond(err, writer)
    })

    handler.HandleFunc("/LogDriver.Capabilities", func(writer http.ResponseWriter, request *http.Request) {
        json.NewEncoder(writer).Encode(&capabilitiesResponse{
            Cap: logger.Capability{ReadLogs: true},
        })
    })

    handler.HandleFunc("/LogDriver.ReadLogs", func(writer http.ResponseWriter, response *http.Request) {
        var req readLogsRequest
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

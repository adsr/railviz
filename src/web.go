package main

import (
    "encoding/json"
    "net"
    "net/http"
    "net/http/fcgi"
)

type FcgiHandler struct {
}

// Handle request
func (s FcgiHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
    if json, err := json.Marshal(trains); err != nil {
        resp.WriteHeader(http.StatusInternalServerError)
    } else {
        resp.Write(json)
    }
}

// Start web server
func startWeb() error {
    if listener, err := net.Listen("tcp", fcgiAddr); err != nil {
        return err
    } else {
        handler := new(FcgiHandler)
        go fcgi.Serve(listener, handler)
        go http.ListenAndServe(httpAddr, handler)
    }
    return nil
}
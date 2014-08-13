package main

import (
    "net"
    "net/http"
    "net/http/fcgi"
    "encoding/json"
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

// Start fcgi server
func startFcgi() error {
    if listener, err := net.Listen("tcp", fcgiAddr); err != nil {
        return err
    } else {
        handler := new(FcgiHandler)
        if err = fcgi.Serve(listener, handler); err != nil {
            return err
        }
    }
    return nil
}

package main

import (
    "encoding/json"
    "net"
    "net/http"
    "net/http/fcgi"
    "fmt"
)

type FcgiHandler struct {
}

// Handle request
func (s FcgiHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
    resp.Header().Set("Content-Type", "application/json")
    if req.URL.RawQuery == "lines" {
        serveLines(resp, req)
    } else {
        serveTrains(resp, req)
    }
}

// Serve trains
func serveTrains(resp http.ResponseWriter, req *http.Request) {
    if json, err := json.Marshal(trains); err != nil {
        resp.WriteHeader(http.StatusInternalServerError)
    } else {
        resp.Write(json)
    }
}

// Serve lines
func serveLines(resp http.ResponseWriter, req *http.Request) {
    type sline struct {
        Color1 string
        Color2 string
        ImageURL string
        Waypoints []*Waypoint
    }
    slines := make(map[string]sline)
    for _, line := range lines {
        slines[line.Id] = sline{
            Color1: line.Color1,
            Color2: line.Color2,
            ImageURL: fmt.Sprintf("http://dummyimage.com/20/%s/%s.gif&text=%s", line.Color1, line.Color2, line.Id[:1]),
            Waypoints: line.Waypoints}
    }
    if json, err := json.Marshal(slines); err != nil {
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

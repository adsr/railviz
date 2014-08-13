package main

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"
)

type routePoint struct {
    *Station
    Coord float64
}

// Read resources files under `path`
func buildResources(path string) error {
    var err error

    // Stations
    stationFiles, err := filepath.Glob(fmt.Sprintf("%s/stations/%s", path, "*"))
    if err != nil {
        return err
    }
    err = buildStations(stationFiles)
    if err != nil {
        return err
    }

    // Lines
    lineFiles, err := filepath.Glob(fmt.Sprintf("%s/lines/%s", path, "*"))
    if err != nil {
        return err
    }
    err = buildServiceLines(lineFiles)
    if err != nil {
        return err
    }

    return nil
}

// Build stations
func buildStations(files []string) error {
    stations = make(map[string]*Station)
    for _, file := range files {
        err := func() error {
            fd, err := os.Open(file)
            if err != nil {
                return err
            }
            defer fd.Close()
            station := new(Station)
            decoder := json.NewDecoder(fd)
            if err = decoder.Decode(station); err != nil {
                return err
            }
            stations[station.Id] = station
            return nil
        }()
        if err != nil {
            return err
        }
    }
    return nil
}

// Build lines
func buildServiceLines(files []string) error {
    lines = make(map[string]*ServiceLine)
    for _, file := range files {
        err := func() error {
            fd, err := os.Open(file)
            if err != nil {
                return err
            }
            defer fd.Close()
            line := new(ServiceLine)
            decoder := json.NewDecoder(fd)
            if err = decoder.Decode(line); err != nil {
                return err
            }
            if err = line.parseRoute(); err != nil {
                return err
            }
            if err = line.parseSched(); err != nil {
                return err
            }
            lines[line.Id] = line
            return nil
        }()
        if err != nil {
            return err
        }
    }
    return nil
}

func (line *ServiceLine) parseSched() error {
    // Parse stops
    dayMins := make([]int, 0, len(line.Stops))
    isPm := false
    lastHour := -1
    var hour, adjHour, min int
    var err error
    for _, stop := range line.Stops {
        hourMin := strings.SplitN(stop, ":", 2)
        if len(hourMin) != 2 {
            return errors.New("Expected h:mm or hh:mm format")
        }
        if hour, err = strconv.Atoi(hourMin[0]); err != nil {
            return errors.New("Expected hour to be a number")
        }
        if min, err = strconv.Atoi(hourMin[1]); err != nil {
            return errors.New("Expected min to be a number")
        }
        adjHour = hour
        if hour == 12 && lastHour == 11 {
            isPm = !isPm
        }
        if !isPm && hour == 12 {
            adjHour = 0
        } else if isPm && hour != 12 {
            adjHour += 12
        }
        lastHour = hour
        dayMins = append(dayMins, adjHour*60+min)
    }

    // Set line.StationStops
    line.StationStops = make([]*StationStop, 0, len(dayMins)*7)
    var prevStationStop *StationStop
    for i := 0; i < len(line.WeeklySched); i += 2 {
        startMin := line.WeeklySched[i]
        endMin := line.WeeklySched[i+1]
        if startMin == -1 || endMin == -1 {
            prevStationStop = nil
            continue
        }
        for j, dayMin := range dayMins {
            if dayMin < startMin && dayMin > endMin {
                continue
            }
            platform := line.Platforms[j%len(line.Platforms)]
            stationStop := &StationStop{
                Platform: platform,
                WeekMin:  dayMin + (i/2)*1440}
            if prevStationStop != nil && j%len(line.Platforms) != 0 {
                prevStationStop.Next = stationStop
            }
            line.StationStops = append(line.StationStops, stationStop)
            prevStationStop = stationStop
        }
    }
    return nil
}

func (line *ServiceLine) parseRoute() error {
    routePoints := []routePoint{}
    for _, route := range line.Route {
        switch val := route.(type) {
        case string:
            if station, ok := stations[val]; !ok {
                return errors.New(fmt.Sprintf("Invalid station id %s", val, line.Id))
            } else {
                routePoints = append(routePoints, routePoint{Station: station})
            }
        case float64:
            routePoints = append(routePoints, routePoint{Coord: val})
        default:
            return errors.New(fmt.Sprintf("Unexpected %T in Waypoints array", val, line.Id))
        }
    }

    // Set line.Waypoints
    var prevStationPlatform *StationPlatform
    for i := 0; i < len(routePoints); i++ {
        routePoint := routePoints[i]
        waypoint := &Waypoint{}
        if routePoint.Station == nil {
            waypoint.Lat = routePoint.Coord
            i += 1
            if i >= len(routePoints) {
                return errors.New("Expected Lon after Lat coordinate")
            }
            waypoint.Lon = routePoints[i].Coord
        } else {
            waypoint.Lat = routePoint.Station.Lat
            waypoint.Lon = routePoint.Station.Lon
            waypoint.Platform = &StationPlatform{
                Station:     routePoint.Station,
                ServiceLine: line,
                Waypoint:    waypoint,
                Prev:        prevStationPlatform}
            waypoint.Platform.Trains = make([]*Train, 0)
            prevStationPlatform = waypoint.Platform
            line.Platforms = append(line.Platforms, waypoint.Platform)
        }
        line.Waypoints = append(line.Waypoints, waypoint)
    }

    fmt.Printf("%+v\n", line.Platforms)

    // Set line.TotalDistance
    var totalDistance float64
    for i, waypoint := range line.Waypoints {
        if i == 0 {
            continue
        }
        totalDistance += line.Waypoints[i].DistanceTo(waypoint)
    }
    line.TotalDistance = totalDistance

    return nil
}

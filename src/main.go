package main

import (
    "flag"
    "fmt"
    "math"
    "time"
)

// Globals
var (
    stations          map[string]*Station
    lines             map[string]*ServiceLine
    trains            []*Train
    simulationMode    bool
    simulationWeekMin int
    simulationSleepMs int
    fcgiAddr          string
    dataDir           string
)

// A station (e.g., WTC)
type Station struct {
    Name string
    Id   string
    Lat  float64
    Lon  float64
}

// A service line (e.g., Newark-WTC)
type ServiceLine struct {
    Name          string
    Id            string
    Color1        string
    Color2        string
    Platforms     []*StationPlatform
    Waypoints     []*Waypoint
    TotalDistance float64
    StationStops  []*StationStop
    Route         []interface{} // For json decode
    WeeklySched   []int         // For json decode
    Stops         []string      // For json decode
}

// A station platform (e.g., the WTC-bound platform @ Grove St on the
// Newark-WTC line)
type StationPlatform struct {
    *Station
    *ServiceLine
    *Waypoint
    Prev   *StationPlatform
    Trains []*Train
}

// A stop at a certain platform (e.g., the weekday 10:21am WTC-bound stop at
// Grove St)
type StationStop struct {
    Platform *StationPlatform
    Next     *StationStop
    WeekMin  int // minute of week (0 thru 10079)
}

// A point on a map with pointers to next/prev points
type Waypoint struct {
    Lat      float64
    Lon      float64
    Platform *StationPlatform
}

// A train
type Train struct {
    Id          int
    CurStop     *StationStop
    CurProgress float64
    Terminated  bool
    Updated     int
    Lat         float64
    Lon         float64
}

// Program entry point
func main() {
    // Flags
    flag.BoolVar(&simulationMode, "s", true, "enable simulation mode")
    flag.IntVar(&simulationSleepMs, "m", 100, "num ms to sleep in simulation mode")
    flag.IntVar(&simulationWeekMin, "w", 0, "weekMin to start at in simulation mode")
    flag.StringVar(&dataDir, "d", "../res", "data directory")
    flag.StringVar(&fcgiAddr, "f", ":9000", "fcgi listen address")
    flag.Parse()
    simulationWeekMin -= 1

    // Parse data
    if err := buildResources(dataDir); err != nil {
        panic(err)
    }

    // Start fcgi server
    if err := startFcgi(); err != nil {
        panic(err)
    }

    // Run trains
    weekMin := 0
    lastWeekMin := 99
    for {
        // TODO No idea what is supposed to happen during DST
        if weekMin = getCurWeekMin(); weekMin != lastWeekMin {
            // Check for station stops that just occurred
            for _, line := range lines {
                for _, stop := range line.GetStationStops(weekMin) {
                    platform := stop.Platform
                    fmt.Printf("[%s] Line %s stop at %s\n", weekMinToStr(weekMin), platform.ServiceLine.Name, platform.Station.Name)
                    // Move train form previous platform this platform
                    var train *Train
                    if train = platform.Prev.DequeueTrain(weekMin); train == nil {
                        // No train was there, so make a new train!
                        train = getOrCreateNewTrain(platform.Station)
                    }
                    train.CurStop = stop
                    train.CurProgress = 0.0
                    train.Terminated = train.CurStop.Next == nil
                    train.Updated = weekMin
                    platform.PushTrain(train)
                }
            }
            lastWeekMin = weekMin
        }

        // Update train progress and position
        updateTrains(weekMin)

        if simulationMode {
            // Sleep a tiny bit
            time.Sleep(time.Duration(simulationSleepMs) * time.Millisecond)
        } else {
            // Sleep a bit
            time.Sleep(1 * time.Second)
        }
    }
}

// Get a `Terminated` train at `station` or create a new one
func getOrCreateNewTrain(station *Station) *Train {
    for _, train := range trains {
        //if train.Terminated && train.CurStop.Platform.Station == station {
        if train.Terminated {
            return train
        }
    }
    train := new(Train)
    train.Id = len(trains)
    trains = append(trains, train)
    fmt.Printf("Spawned new train #%d at station %s\n", train.Id, station.Name)
    return train
}

// Get current minute of week, or increment simulationWeekMin in simulationMode
func getCurWeekMin() int {
    if simulationMode {
        simulationWeekMin += 1
        if simulationWeekMin >= 10080 {
            simulationWeekMin = 0
        }
        return simulationWeekMin
    } else {
        hour, min, _ := time.Now().Clock()
        return hour*60 + min
    }
}

// Get difference between two `weekMin` values, compensating for the 11:59pm
// to 12:01am case
func weekMinDiff(future, now int) int {
    if future < now {
        future += 60 * 24 * 7
    }
    return future - now
}

// Update progress of trains
func updateTrains(weekMin int) {
    for _, train := range trains {
        if !train.Terminated {
            // Update progress til next stop
            train.CurProgress = float64(weekMinDiff(weekMin, train.CurStop.WeekMin)) /
                float64(weekMinDiff(train.CurStop.Next.WeekMin, train.CurStop.WeekMin))
            // Update position
            train.updatePosition()
        }
    }
}

// Update a train's geographical position
func (train *Train) updatePosition() {
    if train.Terminated {
        return
    }
    line := train.CurStop.Platform.ServiceLine
    targetDistance := train.CurProgress * line.TotalDistance

    distanceTally := 0.0
    for i := 0; i < len(line.Waypoints)-1; i += 1 {
        waypoint := line.Waypoints[i]
        nextWaypoint := line.Waypoints[i+1]
        waypointDistance := waypoint.DistanceTo(nextWaypoint)
        if distanceTally+waypointDistance >= targetDistance {
            baseDistance := distanceTally
            distanceTally += waypointDistance
            factor := (targetDistance - baseDistance) / (distanceTally - baseDistance)
            // TODO This does not take into account Earth curvature
            train.Lat = (nextWaypoint.Lat + waypoint.Lat) * factor
            train.Lon = (nextWaypoint.Lon + waypoint.Lon) * factor
            return
        }
    }

    // This should not happen
    train.Lat = train.CurStop.Next.Platform.Waypoint.Lat
    train.Lon = train.CurStop.Next.Platform.Waypoint.Lon
}

// Dequeue a train from `platform`. Only consider trains that have an
// `Updated` value less than `weekMin`.
func (platform *StationPlatform) DequeueTrain(weekMin int) *Train {
    if platform == nil || len(platform.Trains) < 1 {
        return nil
    }
    indexToPop := -1
    for i := len(platform.Trains) - 1; i >= 0; i-- {
        train := platform.Trains[i]
        if train.Updated < weekMin {
            indexToPop = i
            break
        }
    }
    if indexToPop != -1 {
        train := platform.Trains[indexToPop]
        platform.Trains = append(platform.Trains[:indexToPop], platform.Trains[indexToPop+1:]...)
        return train
    }
    return nil
}

// Queue a train to `platform`
func (platform *StationPlatform) PushTrain(train *Train) {
    platform.Trains = append([]*Train{train}, platform.Trains[:]...)
    fmt.Printf("Train %d now at %s\n", train.Id, platform.Station.Name)
}

// Get station stop that occur at `weekMin`
func (line *ServiceLine) GetStationStops(weekMin int) []*StationStop {
    stops := make([]*StationStop, 0)
    for _, stop := range line.StationStops {
        // TODO Sort stops and use binary search
        if stop.WeekMin == weekMin {
            stops = append(stops, stop)
        }
    }
    return stops
}

// Return distance between `a` and `b`
func (a *Waypoint) DistanceTo(b *Waypoint) float64 {
    // TODO This does not take into account Earth curvature
    return math.Sqrt(
        math.Pow(b.Lat-a.Lat, 2) +
            math.Pow(b.Lon-a.Lon, 2))
}

// Convert weekMin to string
func weekMinToStr(weekMin int) string {
    day := weekMin / 1440
    dayMin := weekMin % 1440
    hour := dayMin / 60
    min := dayMin % 60
    return fmt.Sprintf("day=%d %d:%02d", day, hour, min)
}

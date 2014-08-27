railviz
=======

railviz is a rail transit visualization program. Given a set of stations,        
transit lines, and timetables, railviz will figure out the estimated positions
of actual trains.

**Structure**

* The `src` directory contains the source code of railviz (Go). railviz        
derives train data and serves it via a single HTTP endpoint, formatted in
JSON.
* The `res` directory contains definitions of transit lines which are used by
the app to figure out train positions. In this repo there are definitions for
the [PATH] rapid transit system, current as of August 2014. These files can
easily be edited or replaced to visualize other transit systems.
* The `htdocs` directory contains a single file `index.html` which consumes
railviz data and displays trains on a Google Map widget.

**History**

I like trains and wanted to visualize them. I was particularly interested in
deriving the number of trains on a track using only timetable data. After some
research, I discovered [GTFS], a feed format made specifically for this type   
of application, however, the actual feed published by [PATH] contained bogus
data. So I ended up copy-pasting timetables into a custom JSON format. (See
the `res` directory.) This was also more fun.

**Demo**

http://goo.gl/nT7Ljk

[PATH]:http://en.wikipedia.org/wiki/Port_Authority_Trans-Hudson
[GTFS]:https://developers.google.com/transit/gtfs/

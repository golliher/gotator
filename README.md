# Overview

Gotator is a program that runs continusouly and rotates content on a
Firefox browser.  The use I had in mind was to control the rotation of
content on an information radiator.

I was not satisified with the status quo of using a browser plugin to 
rotate on a evenly distributed schedule. For some content 10 seconds is 
enough time.  For other content, 2 minutes might not be enough.

I also wanted to make it easier to 
switch out content schedules remotely.

Features:
* Plays a list of URLs each for the specified amount of time.
* Has a HTTP api for pausing, skipping or playing a URL immediately on screen

Future?
* Have content dependant on time of day or day of week.  i.e.  Monday morning content
might be different than Wednesday afternoon, etc.
* Authentication
* Encrypt communications

# Install

## Dependencies
Gotator has two dependencies:
1. Firefox
2. [FF-Remote-Control (a FireFox extension)](https://github.com/FF-Remote-Control/FF-Remote-Control/releases)

## Source release

With a properly configured Go(lang) environment you can execute Gorotate with 
    go run main.go

## Binary releases

Look in the releases tab of Github and you will find pre-compiled binaries for MacOS and Linux

# Configuration

config.yaml is the configuration for gorotator itself.
default.csv defines the content you want show by gotator

## config.yaml

```IP```            This is the IP address for the FireFox browser that gorotator will control with FF-Remote-Control.

```PORT```          Likewise this is the PORT that the FF-Remote-Control is configured to listen to.

```program_file```  A CSV file containing a gotator program list.   Gotator ships with a configuration to use default.csv

Note:  You can edit config.yaml and change the program_file without interrupting the running gotator.   Gotator watches for 
changes to config.yaml and pulls them in without a restart.  This can be useful for swapping out program files for a running
gotator. 

Also note, go-rotator itself listens for HTTP requests.  At present
these requests are not authenticated and the port is not configurable.   The port is 8080.

Pull requests are welcome.  I intend to improve both when I get the time.  I work on this only over morning coffee.

## default.csv   (or whatever you call your CSV in program_file in config.yaml

Presently the CSV has two fields.  The first is the URL you want displayed.  The second is the duration you would like the content 
displayed before new content rotates in.  Duration can be anything that Golangs time.Duration() understands.    Commonly you will
have content up for minutes or seconds. i.e. "5m" or "30s"  would be valid durations.   "30" or "5" would not.

Changes to CSV files will be picked up at the top of the rotation.   This is because gotator load the program file, displays every
item in it and only then loops back and re-reads the program file.

	
# Usage

The original use for gotator was to run scheduled content without further user input.   A rudimentary HTTP API has since been added that 
allows for more hands on control.

It would be reasonable and expected for you to run gotator in either GNU screen to tmux.    Any console input will be interrepted
as a request to skip past the current content and show the next thing in rotation.

## HTTP Control

For purposes of illustration, assume your IP addres is 127.0.0.1.  Remember that go-rotator listens on port 8080.

### Pause

```http://127.0.0.1:8080/pause``` -  Stop content rotation.

### Resume

```http://127.0.0.1:8080/resume``` - Releases the pause.  Content rotation picks up from where it was paused.

### Skip

```http://127.0.0.1:8080/skip```   - Forces a content rotation early.  If the program calls for the URL to be displayed for 15minutes, using skip will 

### Play

```http://127.0.0.1:8080/play?url='http://example.com'&duration='30s'``` - Pauses normal rotation, shows the requested URL immediately for the specified duration

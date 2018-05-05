# Overview

Gotator is a program that runs continusouly and rotates content on a
browser.  Firefox and Chrome are supported.  The use I had in mind
was to control the rotation of content on an information radiator.

I was not satisified with the status quo of using a browser plugin to 
rotate on a evenly distributed schedule. For some content 10 seconds is 
enough time.  For other content, 2 minutes might not be enough.

I also wanted to make it easier to 
switch out content schedules remotely.

Features:

* Plays a list of URLs each for the specified amount of time.
* Has a HTTP api for pausing, skipping or playing a URL immediately on screen

# Install

## Dependencies
Gotator can control Firefox or Chrome.  You will need one or the other and you will need to start them
with either [Marionette](https://firefox-source-docs.mozilla.org/testing/marionette/marionette/index.html) for Firefox or
remote debugging enabled for Chrome.

### Firefox example (Mac)
```open -n -a Firefox.app --args --marionette -P Automation```

### Chrome example (Mac)
```(cd "/Applications/Google Chrome.app/Contents/MacOS" && ./Google\ Chrome --remote-debugging-port=9222)```

## Binary releases

NEW as of 0.0.5: I'm trying out equinox.io to make it easier for you to install gotator.  Let me know what you think.
https://dl.equinox.io/darrell_golliher/gotator/stable

Older releases are in the releases tab here on Github. You will find pre-compiled binaries for MacOS and Linux.


## Source release

With a properly configured Go(lang) environment you can execute Gorotate with 
```go run main.go```

# Configuration

config.yaml is the configuration for gorotator itself.

default.csv defines the content you want show by gotator

## config.yaml

```BROWSER_IP```            This is the IP address for the FireFox browser that gorotator will control with FF-Remote-Control.

```BROWSER_PORT```          Likewise this is the PORT that Firfox Marionette or Chrome are listenting to.  Hint: It's probably 9222

```BROWSER_MODE``` As of version 0.1.0 Gotator supports three operating modes.

	1 = original FF-remote-control plugin (deprecated)
	2 = Fireforx Marionette 
	3 = Chrome debugging protocol

```program_file```  A CSV file containing a gotator program list.   Gotator ships with a configuration to use default.csv

```apienabled```  Set to 'true' to have gotator listen on HTTP and process REST API requests.  Defaults to false.

```GOTATOR_PORT``` The port Gotator will listen on for API requests.  Defaults to 8080

```timeroverlay``` Set to 'true' to have gotator inject an overlay on top of pages show signal when program is about to change.

Note:  You can edit config.yaml and change the program_file without interrupting the running gotator.   Gotator watches for 
changes to config.yaml and pulls them in without a restart.  This can be useful for swapping out program files for a running
gotator. 

Also note, go-rotator itself listens for HTTP requests.

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
as a request to skip past the current content and show the next thing in rotation if you have set ```interactive: true``` in your config file.

gotator accepts one command line argument.   Running "gotator version" will print the version and exit.  Normal operation is to simply run
"gotator".

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

# Future?

* Have content dependant on time of day or day of week.  i.e.  Monday morning content
might be different than Wednesday afternoon, etc.
* Authentication
* Encrypt communications



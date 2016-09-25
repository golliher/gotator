package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Program struct {
	URL      string
	Duration time.Duration
}

// Loads a list of programs.
// A program consists of a list things to display on the rotator along
// with a number of seconds to display each one before moving on.
func loadProgramList() []Program {
	var list []Program

	bytes, err := ioutil.ReadFile("default.csv")
	if err != nil {
		log.Fatal(err)
	}
	webpages := string(bytes)

	r := csv.NewReader(strings.NewReader(webpages))

	for {
		var p Program
		record, err := r.Read()
		if err == io.EOF {
			fmt.Printf("Finished loading programs.\n\n")
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		p.URL = record[0]
		p.Duration, err = time.ParseDuration(record[1])
		fmt.Printf("Loaded program %s to show for %s.\n", p.URL, p.Duration)
		list = append(list, p)
	}

	return list
}

func runProgram(program Program) {

	ip := viper.Get("IP")
	port := viper.GetInt("PORT")

	constr := fmt.Sprintf("%s:%d", ip, port)

	conn, err := net.Dial("tcp", constr)
	if err != nil {
		fmt.Printf("Error making network connection to: %s\n", constr)
		fmt.Println("Pausing for 30s")
		time.Sleep(30 * time.Second) // wait 30 seconds to slow retries
		return
	}

	fmt.Printf("Running program: %s for %s\n", program.URL, program.Duration)
	fmt.Fprintf(conn, "window.location='%s'\n", program.URL)
	status, err := bufio.NewReader(conn).ReadString('\n')
	fmt.Printf("%s", status)
	time.Sleep(program.Duration)
}

// InitializeConfig loads our configuration using Viper package.
func InitializeConfig() {

	viper.SetConfigType("yaml")
	// Set config file
	viper.SetConfigName("config")

	// Add config path
	//	viper.AddConfigPath("$HOME/.gorotator")
	viper.AddConfigPath(".")

	// Read in the config
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	// Load default settings
	viper.SetDefault("debug", false)

	viper.SetEnvPrefix("gorotator") // will be uppercased automatically
	viper.BindEnv("debug")
	viper.BindEnv("ip")
	viper.BindEnv("port")

	// Do some flag handling and any complicated config logic
	if !viper.IsSet("ip") || !viper.IsSet("port") {
		fmt.Println("Configuration error.  Both IP and PORT must be set via either config or environment.")
		os.Exit(1)
	}

}

func main() {

	InitializeConfig()

	for {
		pl := loadProgramList()
		for _, p := range pl {
			runProgram(p)
		}
		fmt.Printf("Looping back to beginning\n\n")
	}
}

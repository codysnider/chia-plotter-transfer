package main

import (
	"fmt"
	"github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"github.com/joho/godotenv"
	"github.com/radovskyb/watcher"
	"golang.org/x/crypto/ssh"
	"log"
	"math/rand"
	"os"
	"regexp"
	"time"
)

var plotterOutputPath string
var farmerHost string
var farmerUsername string
var farmerKey string
var farmerPlotPath string

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func transferToFarmer(srcPath string) {

	rand.Seed(time.Now().UnixNano())

	clientConfig, _ := auth.PrivateKey(farmerUsername, farmerKey, ssh.InsecureIgnoreHostKey())
	client := scp.NewClientWithTimeout(farmerHost + ":22", &clientConfig, time.Hour)
	err := client.Connect()
	if err != nil {
		log.Println("Couldn't establish a connection to the remote server ", err)
		return
	}

	f, _ := os.Open(srcPath)
	defer client.Close()
	defer f.Close()

	filename := fmt.Sprintf("plot-%s.plot.xfer", randSeq(32))
	dstPath := fmt.Sprintf("%s%s", farmerPlotPath, filename)

	log.Printf("Starting transfer: %s -> %s \n", srcPath, dstPath)

	err = client.CopyFile(f, dstPath, "0655")
	if err != nil {
		log.Println("Error while copying file ", err)
		return
	}
	log.Printf("File transferred: %s -> %s \n", srcPath, dstPath)

	e := os.Remove(srcPath)
	if e != nil {
		log.Fatal(e)
	}
	log.Printf("File deleted: %s\n", srcPath)
}

func monitorForPlotFiles() {
	w := watcher.New()
	w.SetMaxEvents(1)
	w.FilterOps(watcher.Create, watcher.Write)

	r := regexp.MustCompile("(?:.*?).plot$")
	w.AddFilterHook(watcher.RegexFilterHook(r, false))

	go func() {
		for {
			select {
			case event := <-w.Event:
				transferToFarmer(event.Path)
			case err := <-w.Error:
				log.Println(err)
			case <-w.Closed:
				return
			}
		}
	}()

	if err := w.AddRecursive(plotterOutputPath); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Plot file watcher started: %s\n", plotterOutputPath)

	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Println(err)
	}
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	plotterOutputPath 	= os.Getenv("PLOTTER_PLOTTER_OUTPUT_PATH")
	farmerHost 			= os.Getenv("PLOTTER_FARMER_HOST")
	farmerUsername 		= os.Getenv("PLOTTER_FARMER_USERNAME")
	farmerKey 			= os.Getenv("PLOTTER_FARMER_KEY")
	farmerPlotPath 		= os.Getenv("PLOTTER_FARMER_PLOT_PATH")

	monitorForPlotFiles()
}
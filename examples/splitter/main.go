package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/einride/lcm-go/pkg/lcmlog"
)

func main() {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fileName := fs.String("file", "", "lcm file to use")
	splitSizeStr := fs.String("splitSize", "", "split file size MB (default: 1000MB)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalf("running main: %v", err)
	}
	if *fileName == "" {
		log.Fatalf("no file given")
	}
	splitSizeMB := uint32(1000)
	if *splitSizeStr != "" {
		n, err := strconv.Atoi(*splitSizeStr)
		if err != nil {
			log.Fatalf("split size given is not integer")
		}
		if n == 0 {
			log.Fatalf("zero split size given")
		}
		splitSizeMB = uint32(n)
	}
	// Read LCM log
	f, err := os.Open(*fileName)
	if err != nil {
		log.Fatalf("opening file, %v", err)
	}
	// Split log
	logScanner := lcmlog.NewScanner(f)
	err = logScanner.SplitWrite(*fileName, splitSizeMB)
	if err != nil {
		log.Fatalf("scanning log %v", err)
	}
	if err := f.Close(); err != nil {
		log.Fatalf("closing file: %v", err)
	}
}

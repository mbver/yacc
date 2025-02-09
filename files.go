package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/format"
	"os"
)

var oflag string // file to store ftable
var vflag string // file to store foutput

var infile string         // name of input file
var finput *bufio.Reader  // input file
var ftable *bufio.Writer  // the generated go source file
var foutput *bufio.Writer // file to write state output

func init() {
	flag.StringVar(&oflag, "o", "y.go", "generated go source file")
	flag.StringVar(&vflag, "v", "y.output", "generated parse table output file")
	stderr = bufio.NewWriter(os.Stderr)
}

// open input file to read and output files to write to
func openup() {
	infile = flag.Arg(0)
	finput = open(infile)
	if finput == nil {
		errorf("cannot open %v", infile)
	}

	foutput = create(vflag)
	if foutput == nil {
		errorf("can't create file %v", vflag)
	}

	ftable = create(oflag)
	if ftable == nil {
		errorf("can't create file %v", oflag)
	}

}

// open input file to read
func open(s string) *bufio.Reader {
	fi, err := os.Open(s)
	if err != nil {
		errorf("error opening %v: %v", s, err)
	}
	return bufio.NewReader(fi)
}

// create file to write (foutput, ftable)
func create(s string) *bufio.Writer {
	fo, err := os.Create(s)
	if err != nil {
		errorf("error creating %v: %v", s, err)
	}
	return bufio.NewWriter(fo)
}

func exit(status int) {
	if ftable != nil {
		ftable.Flush()
		ftable = nil
		gofmt()
	}
	if foutput != nil {
		foutput.Flush()
		foutput = nil
	}
	if stderr != nil {
		stderr.Flush()
		stderr = nil
	}
	os.Exit(status)
}

func gofmt() {
	src, err := os.ReadFile(oflag)
	if err != nil {
		return
	}
	src, err = format.Source(src)
	if err != nil {
		return
	}
	os.WriteFile(oflag, src, 0666)
}

func usage() {
	fmt.Fprintf(stderr, "usage: yacc [-o go_src_file] [-v parse_output_file] input_file\n")
	exit(1)
}

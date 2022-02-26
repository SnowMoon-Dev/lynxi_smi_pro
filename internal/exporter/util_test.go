package exporter

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
)

var (
	queryTest string
)

func RunShellCmd(command string, arg ...string) (string, error) {
	cmd := exec.Command(command, arg...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_, err := cmd.Stdout.Write([]byte(queryTest))
	if err != nil {
		panic(err)
	}
	out := stdout.String()
	return out, nil
}

func RunShellCmdAndReadLines(fn func(string), command string, arg ...string) {
	f, err := os.Open("/home/lynxi_smi_pro/cmd/lynxi-smi-pro/_" + command)
	if err != nil {
		panic(err)
	}
	r := bufio.NewReaderSize(f, 4*1024)
	line, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		s := string(line)
		fn(s)
		line, isPrefix, err = r.ReadLine()
	}
	if isPrefix {
		log.Infoln("buffer size to small")
		return
	}
	if err != io.EOF {
		fmt.Println(err)
		return
	}
}

func RunShellCmdAndReadStringsbyRef(command string, arg ...string) (*bufio.Reader, *os.File) {
	f, err := os.Open("./lynxi-smi-pro/_" + command)
	if err != nil {
		panic(err)
	}
	r := bufio.NewReader(f)
	return r, f
}

func RunShellCmdAndArgsAndReadStrings(fn func(string), command string, arg ...string) {
	f, err := os.Open("./lynxi-smi-pro/_" + command)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	r := bufio.NewReader(f)
	line, err := r.ReadString('\n')
	for err == nil {
		fn(line)
		line, err = r.ReadString('\n')
	}
	if err != io.EOF {
		fmt.Println(err)
		return
	}

}

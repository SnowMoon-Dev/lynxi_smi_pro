package exporter

import (
	"bufio"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func RunShellCmdGetVersionInfo(fn func(string), command string, arg ...string) {
	r, cmd := runFindVersionInfoCommand(command, arg...)
	line, err := r.ReadString(__LINE_FEED_SEP__)
	for err == nil {
		if strings.Contains(line, "ERROR") || strings.Contains(line, "lynSmi.cpp") || strings.Contains(line, __SN_STR__) || strings.Contains(line, __START_STR__) {
			line, err = r.ReadString(__LINE_FEED_SEP__)
		} else {
			break
		}
		line, err = r.ReadString(__LINE_FEED_SEP__)
	}
	for {
		if err != nil || io.EOF == err {
			break
		}
		fn(line)
		line, err = r.ReadString(__LINE_FEED_SEP__)
	}
	if err != io.EOF {
		fmt.Println(err)
		return
	}
	err = cmd.Wait()
	if err != nil {
		return
	}
}

func RunShellCmdAndArgsAndReadString(fn func(string), command string, arg ...string) {
	cmd := exec.Command(command, arg...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_, err := fmt.Fprintln(os.Stderr, "error=>", err.Error())
		if err != nil {
			return
		}
	}
	err = cmd.Start()
	if err != nil {
		return
	}
	reader := bufio.NewReader(stdout)
	for {
		line, err2 := reader.ReadString(__LINE_FEED_SEP__)
		if err2 != nil || io.EOF == err2 {
			break
		}
		fn(line)
	}
	err = cmd.Wait()
	if err != nil {
		return
	}
}

func RunLynSMICmdAndReadStrings(fn func(string)) {
	r, cmd := runLynSMICommand()
	line, err := r.ReadString(__LINE_FEED_SEP__)
	removeDebugInfo(&line, r, err)
	for {
		if err != nil || io.EOF == err {
			break
		}
		fn(line)
		line, err = r.ReadString(__LINE_FEED_SEP__)
	}
	if err != io.EOF {
		fmt.Println(err)
	}
	err = cmd.Wait()
	if err != nil {
		return
	}
}

func RunShellCmdAndReadStringByRef(command string, arg ...string) (*bufio.Reader, *exec.Cmd) {
	cmd := exec.Command(command, arg...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_, err := fmt.Fprintln(os.Stderr, "error=>", err.Error())
		if err != nil {
			return nil, nil
		}
	}
	err = cmd.Start()
	if err != nil {
		return nil, nil
	}
	r := bufio.NewReader(stdout)
	return r, cmd
}

func ReadLine(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Infoln(err)
		}
	}(f)
	r := bufio.NewReaderSize(f, 4*1024)
	line, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		s := string(line)
		fmt.Println(s)
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

func ReadString(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Infoln(err)
		}
	}(f)
	r := bufio.NewReader(f)
	line, err := r.ReadString(__LINE_FEED_SEP__)
	for err == nil {
		fmt.Print(line)
		line, err = r.ReadString(__LINE_FEED_SEP__)
	}
	if err != io.EOF {
		fmt.Println(err)
		return
	}
}

func ReadStringByFN(filename string, fn func(string)) {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Infoln(err)
		}
	}(f)
	r := bufio.NewReader(f)
	line, err := r.ReadString(__LINE_FEED_SEP__)
	for err == nil {
		fn(line)
		line, err = r.ReadString(__LINE_FEED_SEP__)
	}
	if err != io.EOF {
		fmt.Println(err)
		return
	}
}

func ReadPciInfoStr(filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Infoln(err)
		}
	}(f)
	r := bufio.NewReader(f)
	line, err := r.ReadString(__LINE_FEED_SEP__)
	return line
}

func toJson(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		log.Infof("to json failed, err:%v\n", err)
	}
	return data
}

func jsonToMap(data []byte) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal(data, &m)
	return m, err
}

func structToMap(v interface{}) map[string]interface{} {
	json_str := toJson(v)
	mapData, err := jsonToMap(json_str)
	if err != nil {
		log.Infof("jsonToMap failed, err:%v\n", err)
	}
	return mapData
}

func runLynSMICommandByBoardId(boardId *int) (*bufio.Reader, *exec.Cmd) {
	return RunShellCmdAndReadStringByRef(DefaultLynSmiCommand, LynSmiDetailInfoCmdParam, LynSmiCardIdCmdParam, strconv.Itoa(*boardId))
}

func runLynSMICommandByChipIDAndBoardId(boardId *int, chipId *int) (*bufio.Reader, *exec.Cmd) {
	return RunShellCmdAndReadStringByRef(DefaultLynSmiCommand, LynSmiDetailInfoCmdParam, LynSmiCardIdCmdParam, strconv.Itoa(*boardId),
		LynSmiChipIdCmdParam, strconv.Itoa(*chipId))
}

func runLynSMICommand() (*bufio.Reader, *exec.Cmd) {
	return RunShellCmdAndReadStringByRef(DefaultLynSmiCommand)
}

func runLynSMIDetailCommand() (*bufio.Reader, *exec.Cmd) {
	return RunShellCmdAndReadStringByRef(DefaultLynSmiCommand, LynSmiDetailInfoCmdParam)
}

func runFindVersionInfoCommand(cmd string, arg ...string) (*bufio.Reader, *exec.Cmd) {
	return RunShellCmdAndReadStringByRef(cmd, arg...)
}

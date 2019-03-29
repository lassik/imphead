// © 2019 Lassi Kortela
// SPDX-License-Identifier: ISC

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

const prog = "imphead"

func main() {
	var n int
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"usage: %s [-n <lines>] <command> [<arg ...>]\n\n"+
				"Impatient head. Show some output from"+
				" <command> then kill it.\n\n",
			os.Args[0])
		flag.PrintDefaults()
	}
	flag.IntVar(&n, "n", 10, "number of lines to show")
	flag.Parse()
	subArgs := flag.Args()
	if len(subArgs) < 1 || n < 1 {
		flag.Usage()
		os.Exit(2)
	}
	subCmd := exec.Command(subArgs[0], subArgs[1:]...)
	//subCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	subCmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	subCmd.Stdin = os.Stdin
	subCmd.Stderr = os.Stderr
	subOutUnbuffered, err := subCmd.StdoutPipe()
	if err != nil {
		die(err)
	}
	subOut := bufio.NewReader(subOutUnbuffered)
	err = subCmd.Start()
	if err != nil {
		die(err)
	}
	nRead := 0
	for {
		bytes, err := subOut.ReadBytes('\n')
		if err != nil && err != io.EOF {
			die(err)
		}
		os.Stdout.Write(bytes)
		nRead++
		if nRead >= n || err == io.EOF {
			break
		}
	}
	processGroupID, err := syscall.Getpgid(subCmd.Process.Pid)
	if err == nil {
		err = syscall.Kill(-processGroupID, syscall.SIGINT)
		if err != nil {
			die("cannot interrupt process group", err)
		}
	}
	err = subCmd.Wait()
	if !isNormalExitOrInterrupt(err) {
		die(err)
	}
}

func isNormalExitOrInterrupt(errFromWait error) bool {
	if errFromWait == nil {
		return true
	}
	exit, isExitError := errFromWait.(*exec.ExitError)
	if !isExitError {
		return false
	}
	if exit.Success() {
		return true
	}
	waitStatus, hasWaitStatus := exit.Sys().(syscall.WaitStatus)
	if !hasWaitStatus {
		return false
	}
	if waitStatus.Signal() == os.Interrupt {
		return true
	}
	return false
}

func die(vs ...interface{}) {
	msg := prog
	for _, v := range vs {
		msg += ": " + fmt.Sprintf("%v", v)
	}
	msg += "\n"
	os.Stderr.Write([]byte(msg))
	os.Exit(1)
}

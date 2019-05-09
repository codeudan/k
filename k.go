package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/manifoldco/promptui"
)

func main() {
	if runtime.GOOS == "windows" {
		os.Exit(1)
	}

	pidToPorts := listPortByPid()
	psList := listProcess(pidToPorts)
	psDisplayList := []string{}
	for _, ps := range psList {
		str := ""
		str = str + ps.Name + " " + "(" + ps.Pid + ")"
		if len(ps.Ports) > 3 {
			port1 := ps.Ports[0]
			portEnd := ps.Ports[len(ps.Ports)-1]
			str = str + " " + port1 + " ... " + portEnd
		} else {
			str = str + " " + strings.Join(ps.Ports, ", ")
		}
		psDisplayList = append(psDisplayList, str)
	}
	prompt := promptui.Select{
		Label:             "",
		Items:             psDisplayList,
		Size:              10,
		StartInSearchMode: true,
		Templates: &promptui.SelectTemplates{
			Label:    "Running processes:",
			Active:   "❯ {{ . }}",
			Inactive: "  {{ . }}",
			Selected: "  {{ . }}",
		},
		Searcher: func(input string, idx int) bool {
			p := psList[idx]
			name := strings.ToLower(p.Name)
			input = strings.ToLower(input)
			if strings.Contains(name, input) {
				return true
			}
			if strings.Contains(p.Pid, input) {
				return true
			}
			for _, port := range p.Ports {
				if strings.Contains(port, input) {
					return true
				}
			}
			return false
		},
	}

	idx, _, err := prompt.Run()
	if err == nil {
		kill(psList[idx].Pid)
	}
}

type ps struct {
	Pid   string
	Comm  string
	Name  string
	Ports []string
}

func listProcess(pidToPorts map[string][]string) []ps {
	cmd := exec.Command("ps", "wwxo", "pid,comm")
	stdout, err := cmd.Output()
	if err != nil {
		os.Exit(1)
	}
	result := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	result = result[1:]
	data := []ps{}
	for _, line := range result {
		line = strings.TrimSpace(line)
		pid := strings.SplitN(line, " ", 2)[0]
		comm := line[len(pid)+1:]
		if strings.HasSuffix(comm, "-helper") ||
			strings.HasSuffix(comm, "Helper") ||
			strings.HasSuffix(comm, "HelperApp") {
			continue
		}
		data = append(data, ps{
			pid,
			comm,
			filepath.Base(comm),
			pidToPorts[pid],
		})
	}
	return data
}

func listPortByPid() map[string][]string {
	osName := runtime.GOOS
	pidToPorts := map[string][]string{}
	cmdOutput := getNetstatOutput()
	lists := strings.Split(strings.TrimSpace(cmdOutput), "\n")
	validLineRegStr := "^\\s*(tcp|udp)"
	itemReg, _ := regexp.Compile("([\\w*.*])+")
	portReg, _ := regexp.Compile("[^]*[.:](\\d+)$")
	for _, line := range lists {
		isValidLine, _ := regexp.MatchString(validLineRegStr, line)
		if !isValidLine {
			continue
		}
		lineData := itemReg.FindAllString(line, -1)
		if osName == "darwin" && len(lineData) < 10 {
			end := append([]string{}, lineData[5:]...)
			start := append(lineData[0:5], "")
			lineData = append(start, end...)
		}
		portData := []string{}
		pid := ""
		if osName == "darwin" {
			portData = portReg.FindAllString(lineData[3], -1)
			pid = lineData[8]
		}
		if osName == "linux" {
			portData = portReg.FindAllString(lineData[4], -1)
			pid = lineData[6]
		}
		if pidToPorts[pid] == nil {
			pidToPorts[pid] = []string{}
		}
		if len(portData) > 0 {
			port := portData[0]
			if !contains(pidToPorts[pid], port) {
				pidToPorts[pid] = append(pidToPorts[pid], port)
			}
		}
	}
	return pidToPorts
}

func getNetstatOutput() string {
	osName := runtime.GOOS
	listStr := ""
	if osName == "darwin" {
		tcpCmd := exec.Command("netstat", "-anv", "-p", "tcp")
		tcp, tcpErr := tcpCmd.Output()
		if tcpErr != nil {
			os.Exit(1)
		}
		udpCmd := exec.Command("netstat", "-anv", "-p", "udp")
		udp, udpErr := udpCmd.Output()
		if udpErr != nil {
			os.Exit(1)
		}
		listStr = string(tcp) + "\n" + string(udp)
	}
	if osName == "linux" {
		cmd := exec.Command("ss", "-tunlp")
		out, outErr := cmd.Output()
		if outErr != nil {
			os.Exit(1)
		}
		listStr = string(out)
	}
	return listStr
}

func kill(pid string) {
	killCmd := exec.Command("kill", pid)
	_, killErr := killCmd.Output()
	if killErr == nil {
		fmt.Println("  Killed " + pid)
		return
	}

	prompt := promptui.Prompt{
		Label:     "Force kill?",
		IsConfirm: true,
	}

	confirm, _ := prompt.Run()
	if confirm == "y" {
		killCmd := exec.Command("kill", "-9", pid)
		killCmd.Output()
		fmt.Println("  Killed " + pid)
	}
}

func contains(list []string, s string) bool {
	for _, elem := range list {
		if elem == s {
			return true
		}
	}
	return false
}

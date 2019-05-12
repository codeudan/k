package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"github.com/manifoldco/promptui"
)

const size int = 10
const portMaxDisplay int = 6

func main() {
	if runtime.GOOS == "windows" {
		fmt.Println("Windows is not supported")
		os.Exit(1)
	}

	pidToPorts := listPortByPid()
	psList := listProcess(pidToPorts)

	prompt := promptui.Select{
		Label:             "",
		Items:             psList,
		Size:              size,
		StartInSearchMode: true,
		Templates: &promptui.SelectTemplates{
			Label:    "Running processes:",
			Active:   `{{"❯ " | cyan }}{{.Name | cyan}} {{ "(" | cyan}}{{ .Pid | cyan }}{{")" | cyan}} {{ .PortsStr | redLight}}`,
			Inactive: `  {{.Name }} ({{ .Pid }}) {{ .PortsStr | red}}`,
			Selected: `  {{.Name }} ({{ .Pid }}) {{ .PortsStr | red}}`,
			FuncMap: template.FuncMap{
				"faint":    promptui.Styler(0, 0, 31),
				"cyan":     promptui.Styler(0, 0, 36),
				"red":      promptui.Styler(2, 40, 35),
				"redLight": promptui.Styler(0, 0, 35),
			},
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
		return
	}
}

type ps struct {
	Pid      string
	Name     string
	Ports    []string
	PortsStr string
}

func listProcess(pidToPorts map[string][]string) []ps {
	cmd := exec.Command("ps", "wwxo", "pid,comm")
	stdout, err := cmd.Output()
	if err != nil {
		errorHandler(err)
	}
	result := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	result = result[1:]
	data := []ps{}
	for _, line := range result {
		line = strings.TrimSpace(line)
		pid := strings.SplitN(line, " ", 2)[0]
		comm := line[len(pid)+1:]
		ports := pidToPorts[pid]
		portsStr := ""
		portsLen := len(ports)
		if portsLen > portMaxDisplay {
			portStart := []string{ports[0], ports[1], ports[2]}
			portEnd := []string{ports[portsLen-3], ports[portsLen-2], ports[portsLen-1]}
			portsStr = strings.Join(portStart, ", ") + " ··· " + strings.Join(portEnd, ", ")
		} else {
			portsStr = portsStr + " " + strings.Join(ports, ", ")
		}
		if strings.HasSuffix(comm, "-helper") ||
			strings.HasSuffix(comm, "Helper") ||
			strings.HasSuffix(comm, "HelperApp") {
			continue
		}
		data = append(data, ps{
			pid,
			filepath.Base(comm),
			pidToPorts[pid],
			portsStr,
		})
	}
	return data
}

func listPortByPid() map[string][]string {
	validNetLineRegStr := "^\\s*(tcp|udp)"
	allFieldsReg, _ := regexp.Compile("\\S+")
	portReg, _ := regexp.Compile(".*[.:](\\d+)$")
	pidReg, _ := regexp.Compile("(?:^|\",|\",pid=)(\\d+)")

	osName := runtime.GOOS
	portPidIdx := map[string]int{}
	if osName == "darwin" {
		portPidIdx = map[string]int{
			"port": 3,
			"pid":  8,
		}
	}
	if osName == "linux" {
		portPidIdx = map[string]int{
			"port": 4,
			"pid":  6,
		}
	}
	pidToPorts := map[string][]string{}
	cmdOutput := getNetstatOutput()
	lists := strings.Split(strings.TrimSpace(cmdOutput), "\n")
	for _, line := range lists {
		isValidLine, _ := regexp.MatchString(validNetLineRegStr, line)
		if !isValidLine {
			continue
		}
		allFields := allFieldsReg.FindAllString(line, -1)
		if osName == "darwin" && len(allFields) < 10 {
			end := append([]string{}, allFields[5:]...)
			start := append(allFields[0:5], "")
			allFields = append(start, end...)
		}
		port := ""
		pid := ""
		portData := portReg.FindStringSubmatch(allFields[portPidIdx["port"]])
		if len(portData) > 0 {
			port = portData[1]
		}
		pidParsed := pidReg.FindStringSubmatch(allFields[portPidIdx["pid"]])
		if len(pidParsed) > 0 {
			pid = pidParsed[1]
		}
		if pid == "" || port == "" {
			continue
		}
		if pidToPorts[pid] == nil {
			pidToPorts[pid] = []string{}
		}
		if !contains(pidToPorts[pid], port) {
			pidToPorts[pid] = append(pidToPorts[pid], port)
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
			errorHandler(tcpErr)
		}
		udpCmd := exec.Command("netstat", "-anv", "-p", "udp")
		udp, udpErr := udpCmd.Output()
		if udpErr != nil {
			errorHandler(udpErr)
		}
		listStr = string(tcp) + "\n" + string(udp)
	}
	if osName == "linux" {
		cmd := exec.Command("ss", "-tunlp")
		out, outErr := cmd.Output()
		if outErr != nil {
			errorHandler(outErr)
		}
		listStr = string(out)
	}
	return listStr
}

func kill(pid string) {
	killCmd := exec.Command("kill", pid)
	_, killErr := killCmd.Output()
	if killErr == nil {
		printKill()
		return
	}
	prompt := promptui.Prompt{
		Label:     "Force kill?",
		IsConfirm: true,
	}
	confirm, _ := prompt.Run()
	if confirm == "y" {
		killCmd := exec.Command("kill", "-9", pid)
		_, killErr := killCmd.Output()
		if killErr == nil {
			printKill()
			return
		}
		errorHandler(killErr)
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

func errorHandler(err error) {
	fmt.Println(err)
	os.Exit(1)
}

func printKill() {
	fmt.Printf(" %c[%d;%d;%dm Killed%c[0m ", 0x1B, 0, 0, 35, 0x1B)
	fmt.Println("")
}

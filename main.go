package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	ServersDir      string `json:"servers_dir"`
	NginxConfigPath string `json:"nginx_config_path"`
	BasePort        int    `json:"base_port"`
	NvmPath         string `json:"nvm_path"`
}

var config Config
var runningProcesses []ServerProcess

const ConfigFile = "config.json"

type ServerProcess struct {
	PID        int    `json:"pid"`
	WorkingDir string `json:"workingDir"`
}

func loadConfig() {
	file, err := os.ReadFile(ConfigFile)
	if err == nil {
		json.Unmarshal(file, &config)
	} else {
		config = Config{
			ServersDir:      "./servers",
			NginxConfigPath: "/etc/nginx/nginx.conf",
			BasePort:        2000,
			NvmPath:         "~/.nvm/nvm.sh",
		}
	}
}

func saveConfig() {
	data, _ := json.MarshalIndent(runningProcesses, "", "  ")
	os.WriteFile(ConfigFile, data, 0644)
}

func runCommand(command, workingDir string, isStart bool) (int, error) {
	fullCommand := command
	if _, err := os.Stat(filepath.Join(workingDir, ".nvmrc")); err == nil {
		fullCommand = fmt.Sprintf("source %s && nvm use && %s", config.NvmPath, command)
	}
	cmd := exec.Command("bash", "-c", fullCommand)
	cmd.Dir = workingDir

	if isStart {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := cmd.Start()
	if err != nil {
		return 0, err
	}

	return cmd.Process.Pid, nil
}

func processServer(serverName string, stop bool, port *int) (string, int, bool, error) {
	serverPath := filepath.Join(config.ServersDir, serverName)
	settingsPath := filepath.Join(serverPath, ".settings")
	realServerPath := filepath.Join(serverPath, "server")
	file, err := os.Open(settingsPath)
	if err != nil {
		return "", 0, false, nil
	}
	defer file.Close()

	settings := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "=", 2)
		if len(parts) == 2 {
			settings[parts[0]] = strings.Trim(parts[1], "\"")
		}
	}

	serverless := settings["SERVERLESS"] == "true"
	if !serverless {
		settings["PORT"] = strconv.Itoa(*port)
	}

	if stop {
		for _, proc := range runningProcesses {
			if proc.WorkingDir == realServerPath {
				exec.Command("kill", strconv.Itoa(proc.PID)).Run()
				exec.Command("pkill", "-TERM", "-P", strconv.Itoa(proc.PID)).Run()
				exec.Command("kill", "-9", strconv.Itoa(proc.PID)).Run()
				fmt.Println("Stopped server:", serverName)
				runningProcesses = removeProcess(proc.PID)
				saveConfig()
			}
		}
		return "", 0, serverless, nil
	}

	if startCmd, exists := settings["START"]; exists {
		pid, err := runCommand(startCmd, realServerPath, true)
		if err == nil {
			runningProcesses = append(runningProcesses, ServerProcess{pid, realServerPath})
			saveConfig()
			fmt.Println("Started server:", serverName)
			return settings["MATCH"], *port, serverless, nil
		}
	}

	if !serverless {
		*port++
	}
	return "", 0, serverless, nil
}

func updateNginxConfig(servers []string) {
	configData := fmt.Sprintf(`user www-data;
worker_processes auto;
pid /run/nginx.pid;
include /etc/nginx/modules-enabled/*.conf;

events {
    worker_connections 768;
}

http {
    large_client_header_buffers 4 16k;
%s
}`, strings.Join(servers, "\n"))
	os.WriteFile(config.NginxConfigPath, []byte(configData), 0644)
	exec.Command("sudo", "systemctl", "restart", "nginx").Run()
	fmt.Println("Nginx configuration updated!")
}

func removeProcess(pid int) []ServerProcess {
	var updated []ServerProcess
	for _, proc := range runningProcesses {
		if proc.PID != pid {
			updated = append(updated, proc)
		}
	}
	return updated
}

func main() {
	loadConfig()
	args := os.Args
	if len(args) < 2 {
		fmt.Println("Usage: sudo serman [start|stop]")
		return
	}

	command := args[1]
	stop := command == "stop"
	port := config.BasePort
	nginxConfig := []string{}
	dirs, _ := os.ReadDir(config.ServersDir)
	for _, dir := range dirs {
		if dir.IsDir() {
			match, srvPort, serverless, err := processServer(dir.Name(), stop, &port)
			if err == nil && match != "" && !serverless {
				nginxConfig = append(nginxConfig, fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    location / {
        proxy_pass http://localhost:%d;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}`, match, srvPort))
			}
		}
	}

	if !stop {
		updateNginxConfig(nginxConfig)
	}
}

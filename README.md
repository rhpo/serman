# Server Manager

## Overview
Server Manager (`serman`) is a tool to manage multiple servers with automatic Nginx configuration and process handling. It allows starting and stopping servers while managing port assignments and reverse proxy settings.

## Requirements
- Go (latest stable version)
- Nginx
- Node.js with NVM
- Linux-based OS (Ubuntu recommended)

## Installation
1. Clone the repository:
   ```sh
   git clone <repo_url>
   cd <repo_name>
   ```
2. Build the binary:
   ```sh
   go build -o serman main.go
   ```
3. Move the binary to `/usr/local/bin/` for global access:
   ```sh
   sudo mv serman /usr/local/bin/
   ```

## Configuration
Create a `config.json` file in the same directory as `serman` with the following structure:
```json
{
  "servers_dir": "./servers",
  "nginx_config_path": "/etc/nginx/nginx.conf",
  "base_port": 2000,
  "nvm_path": "~/.nvm/nvm.sh"
}
```

### Parameters:
- `servers_dir`: Directory containing the server projects.
- `nginx_config_path`: Path to the Nginx configuration file.
- `base_port`: Starting port number for the servers.
- `nvm_path`: Path to the NVM initialization script.

## Usage
Run the following command to start all servers:
```sh
sudo serman start
```
To stop all running servers:
```sh
sudo serman stop
```

## Server Configuration
Each server should have a `.settings` file in its root directory with the following structure:
```
START="npm start"
MATCH="example.local"
SERVERLESS="false"
```
- `START`: Command to start the server.
- `MATCH`: Domain name to be used in Nginx.
- `SERVERLESS`: Set to `true` if no backend service is needed.

## Nginx Integration
When starting servers, `serman` will generate an Nginx configuration automatically. Ensure Nginx is installed and running:
```sh
sudo systemctl restart nginx
```

## License
This project is licensed under the MIT License.


<p align="center">
  <img width="150" height="74" src="https://user-images.githubusercontent.com/6007737/124795490-5fa5a100-df50-11eb-9483-0d8aa2ac09f6.png">
</p>

# Golang Preprocessor for Web Services - GOP Server #


## Introduction ###

The GOP Server is an open-source web server for Linux and Windows systems that handles HTTP requests to scripts programmed in Go returning the result of the execution to the client.

The software is able to, depending on the requested path, return static files or execute Go scripts, which are compiled on the fly and have specific functionality. This functionality is transparently embedded by the GOP Server into the script files to provide them with functions inherent to web application development such as URL parameter management, cookie management, session management, includes a library of basic functions and database support.

GOP scripts are .gop files coded in Go but that follow the GOP specification, a superset of Go defined to integrate the extra functionality provided by the server.

In summary, the server allows to easily develop web applications programmed in Go with the comfort and functionality analogous to PHP, which makes it a multipurpose tool capable of being used both in educational environments to learn the language and in development and production environments for web applications or HTTP APIs.

## Disclaimer ###

*The software has just been released an has to be considered as beta, as might contain bugs and/or security vulnerabilities. Audit the code before using it in production enviroments. The software is provided as is without warranty of any kind.*

## How to run ###
**The steps are simple: install Golang, clone the repository and execute from source (go run).**

Then, place your files under the /public directory and access them from http://localhost:80.

### Windows Prerequisites
Install Golang from https://golang.org/doc/install

Clone the repository.

Edit the configuration if needed (config/configuration.json)

Open a terminal inside the GOP-Server folder.

### Linux Prerequisites
```
sudo apt update
sudo apt install golang
```
```
git clone https://github.com/i-rme/GOP-Server.git
cd GOP-Server
nano config/configuration.json
```

### Linux
```sudo go run src/main.go```

### Windows
```go run src/main.go```

### WSL
```wsl sudo go run src/main.go```

Root is only needed to listen on port 80. With a `Port` above 1024, `go run src/main.go` is enough.

To use a configuration file somewhere else, pass its path: ```go run src/main.go -config /etc/gop/configuration.json```

## Configuration
Every key in `config/configuration.json` is optional; a missing key takes its default from `src/server/config/config.go`, which matches the shipped file except that there are no aliases or denied ranges and logging is at the `basic` level. Timeouts can be written in nanoseconds, as in the shipped file, or as strings like `"12s"` or `"1m"`. A configuration with a bad value (an invalid `DenyFrom` range, a missing `DocumentRoot`, an unknown `LoggingLevel`) stops the server at startup with a message naming the key.

Set *GopKeySalt* to a long random string. While it keeps the placeholder value, the server picks a random one at each start and logs a warning.

Scripts are compiled with `go build` the first time they are requested (`X-Cache-Status: MISS`) and the binary is reused until the script, or a file it requires, changes (`HIT`). Compiling is limited by *BuildTimeout* (default `"60s"`) and each run by *ExecutionTimeout* (default `"6s"`); a script that runs too long gets a timeout page and is killed together with any process it started.

## Optional Settings
Enable database support by setting *MySQLSupportEnabled* to `true` in the config file. The driver is fetched by the go command the first time a script is built, so no `go get` is needed.

Enable privilege downgrade by setting *RunScriptsAsNobody* to `true` in the config file. Scripts are then built and run as the `nobody` user. This needs the server to run as root on Linux or another Unix system; elsewhere the server refuses to start rather than run scripts with its own privileges.

## Important paths
**Configuration file**: `./config/configuration.json`

**Default document root**: `./public`

**Default log directory**: `./logs`

**Default uploads directory**: `./private/uploads`

## Troubleshooting
**Error**: *CreateFile src/main.go: The system cannot find the path specified.* when launching the GOP Server with go run.

**Solution**: Navigate to the GOP-Server directory before launching. ```cd GOP-Server```

**Error**: "*listen tcp 127.0.0.1:80: bind: permission denied*" when starting the server on linux.

**Solution**: Run the server with root privileges ```sudo go run src/main.go```, or set *Port* above 1024.


## Architecture
![Architecture](https://user-images.githubusercontent.com/6007737/124795367-3f75e200-df50-11eb-9e4b-4cd9171f8bc5.png)

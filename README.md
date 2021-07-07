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

### Windows Prerequisites
Install Golang from https://golang.org/doc/install

### Linux Prerequisites
```
sudo apt update
sudo apt install golang
```
```
git clone https://github.com/i-rme/GOP-Server.git
cd pfg
nano config/configuration.json
```

### Linux
```sudo go run src/main.go```

### Windows
```go run src/main.go```

### WSL
```wsl sudo go run src/main.go```

## Optional Settings
Enable database support by setting *MySQLSupportEnabled* to `true` in the config file.

Enable privilege downgrade by setting *RunScriptsAsNobody* to `true` in the config file.

## Important paths
**Configuration file**: `./config/configuration.json`

**Default document root**: `./public`

**Default log directory**: `./logs`

**Default uploads directory**: `./private/uploads`

**CMS Example database details**: `./public/examples/cms/components/repository.gop`

## Troubleshooting
**Error**: "*no required module provides package*" when MySQL support is enabled.

**Solution**: Install the missing package using ```go get github.com/go-sql-driver/mysql```

**Error**: "*listen tcp 127.0.0.1:80: bind: permission denied*" when starting the server on linux.

**Solution**: Run the server with root privileges ```sudo go run src/main.go```


## Architecture
![Architecture](https://user-images.githubusercontent.com/6007737/124795367-3f75e200-df50-11eb-9e4b-4cd9171f8bc5.png)

# README #

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

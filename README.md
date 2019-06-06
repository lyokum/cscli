# Cscli
Unofficial command line interface for University of Notre Dame's Class Search site

## Disclaimer
*This project is in the very early stages of development and is liable to big changes in the future. Do not expect stability for the time being.*

## Contributions
Please email me at `lyokum@nd.edu` if you would be interested in contributing or have ideas about improvements. You can also open up an Issue if you have questions or problems.

## Installation
```sh
pacman -S go

go get github.com/mattn/go-isatty
go get github.com/urfave/cli

go get golang.org/x/net/html

go get github.com/lyokum/attr
go get github.com/lyokum/mail-send
go get github.com/lyokum/update
go get github.com/lyokum/cscli

go install github.com/lyokum/cscli
```

Also, add the bin/ directory of your $GOPATH to your path:
```sh
export PATH=$PATH:$(go env GOPATH)/bin
```

To use the update functionality, you will need to have a notification server running:
```sh
pacman -S notification-daemon
pacman -S dunst

dunst &
```

## Current plans for the future
- Make a graphical frontend
- Add config files for easier use
- Add user config file and add parsing for class pages to check if user fits class requirements

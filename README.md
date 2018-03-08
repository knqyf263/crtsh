# crtsh
API client for crt.sh

`crtsh` allows to get the information about a HTTPS website.  
e.g. Search subdoimains.

This tool uses Certificate Transparency logs.
For more information, check https://crt.sh/

## Example
### Subdomains
Search subdomains of `example.com`

<img src="img/subdomains.png">

### Organization
In `--query`, `%` can be used as wildcard.

<img src="img/query.png">



## Install
### Binary
Download binary from [release page](https://github.com/knqyf263/crtsh/releases)

### Source
```
$ go get -u github.com/knqyf263/crtsh
```



## Usage

```
% crtsh -h
crtsh client

Usage:
  crtsh [command]

Available Commands:
  help        Help about any command
  search      search

Flags:
      --config string   config file (default is $HOME/.crtsh.yaml)
  -h, --help            help for crtsh

Use "crtsh [command] --help" for more information about a command.
```

### search
```
$ crtsh search --help
Usage:
  crtsh search [flags]

Flags:
  -d, --domain string   Domain Name (e.g. %.exmaple.com)
  -h, --help            help for search
      --plain           plain text mode
  -q, --query string    query (e.g. Facebook)

Global Flags:
      --config string   config file (default is $HOME/.crtsh.yaml)

```

# proto-gopackage
Tool to set or replace the go_package option in all protobuf files of a directory, according to a user-specified rule

Suppose you receive a bunch of [gRPC](https://grpc.io/) .proto files and your task is to write a [Go](https://golang.org/) client for it. You will observe that it is tricky to generate correct Go code from the .proto files. Especially when the original Protobuf files import each other. That's because each .proto file needs to declare `option go_package = "..."` in a way that is consistent with Go's module conventions. But most Protobuf files out there either have no such option or set it to an import path that doesn't match your module name.

**proto-gopackage** solves this problem by enabling you to manage `option go_package` declarations consistently across all .proto files within a directory structure. First, it will match their `package` declaration against a regular expression, then it will add or update the `option go_package` directive according to a template.

# Installation
Install using `go get`:
```
> go get -u github.com/indyjo/proto-gopackage
```
This will place `proto-gopackage` in your `~/go/bin` folder.

# Example
Suppose you have a Go module named `acme.com/client`. Beneath its main directory, there is s folder named `proto/` which contains a bunch of proto files, some of those provided by your company, *Acme Corp.*. Those .proto files haven't been prepared for Go, so they're lacking proper `option go_package` declarations.

```
> proto-gopackage -package 'acme\.(.*)' -go_package 'acme.com/client/proto/acme/{{index . 1}}' ./proto
```
The above command will scan through folder `./proto`, setting (or replacing) `option go_package` in all .proto files whose `package` declaration starts with `acme.`. Notice that package substrings containing a dot `.` will automatically be converted to slash `/` when inserted into `option go_package`.

# Versioned packages
Some .proto files have versioned packages, that is they declare a `package` suffixed by a version. Something like `acme.logging.v2`. You might want those packages to receive an `option go_package` in extended format like `acme.com/client/proto/logging/v2;logging`. Easy with the following command:
```
> proto-gopackage -package 'acme\.(.*)\.([^.]+)\.(v.)' -go_package 'acme.com/client/proto/acme/{{index . 1}}/{{index . 2}}/{{index . 3}};{{index . 2}}' ./proto
```

module github.com/DotNetAge/goreact

go 1.25.0

replace (
	github.com/DotNetAge/gochat => ../gochat
	github.com/DotNetAge/gograph => ../gograph
	github.com/DotNetAge/gorag => ../gorag
)

require (
	github.com/DotNetAge/gochat v0.0.0-00010101000000-000000000000
	github.com/emersion/go-imap v1.2.1
	github.com/emersion/go-message v0.18.2
)

require (
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21 // indirect
	golang.org/x/text v0.35.0 // indirect
)

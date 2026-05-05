module github.com/DotNetAge/goreact

go 1.26

replace (
	github.com/DotNetAge/gochat => ../gochat
	github.com/DotNetAge/gograph => ../gograph
	github.com/DotNetAge/gorag => ../gorag
)

require (
	github.com/DotNetAge/gochat v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.3.0
	github.com/pkoukk/tiktoken-go v0.1.8
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/dlclark/regexp2 v1.10.0 // indirect

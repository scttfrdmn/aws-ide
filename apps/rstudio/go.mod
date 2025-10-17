module github.com/scttfrdmn/aws-ide/apps/rstudio

go 1.22

replace github.com/scttfrdmn/aws-ide/pkg => ../../pkg

require (
	github.com/scttfrdmn/aws-ide/pkg v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.8.0
	gopkg.in/yaml.v3 v3.0.1
)

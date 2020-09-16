# Depoy

## Execute Sonarqube

- cd src
- go test ./... -coverprofile="coverage-report.out"
- go test ./... -json > test-report.out

```properties
sonar.projectKey=depoy
sonar.sourceEncoding=UTF-8
sonar.host.url=http://localhost:9000
sonar.exclusions=**/*_test.go,**/vendor/**
sonar.test.inclusuins=**/*_test.go
sonar.test.exclusions=**/vendor/**
sonar.sources=.
sonar.go.tests.reportPaths=test-report.out
sonar.go.coverage.reportPaths=coverage-report.out
```

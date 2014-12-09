# github-auth

GitHub authentication library for golang.

Documentation: [![GoDoc](https://godoc.org/github.com/satococoa/github-auth?status.svg)](https://godoc.org/github.com/satococoa/github-auth)
Build Status: [![Circle CI](https://circleci.com/gh/satococoa/github-auth.svg?style=svg)](https://circleci.com/gh/satococoa/github-auth)

## Usage

```go
scopes  := []string{"repo", "public_repo", "read:org"}
appName := "YourAppName"
client := client.CreateClient(appName, scopes)
```

If appName is "awesomeapp", this library will make ~/.awesomeapp.conf to store "Personal access token".

Also, your "Personal access token" that appears in [Applications](https://github.com/settings/applications) page will be named "awesomeapp".

## License

github-auth is available under the MIT license. See the LICENSE file for more info.

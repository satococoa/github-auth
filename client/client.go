package client

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"regexp"
	"strings"

	"code.google.com/p/goauth2/oauth"
	"github.com/google/go-github/github"
	"github.com/howeyc/gopass"
	"github.com/jmcvetta/napping"
)

// CreateClient は認証済みの github.Client を返します
func CreateClient(appName string, scopes []string) *github.Client {
	// 設定ファイルがあれば読み込む
	var accessToken string
	var err error

	user, _ := user.Current()
	confName := "." + strings.ToLower(appName) + ".conf"
	confpath := user.HomeDir + "/" + confName
	buf, err := ioutil.ReadFile(confpath)
	if err != nil {
		// 認証して accessToken を取得する
		accessToken, err = fetchAccessToken(appName, scopes)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		ioutil.WriteFile(confpath, []byte(accessToken), 0644)
	} else {
		// ファイルから accessToken を読み込む
		accessToken = fmt.Sprintf("%s", buf)
	}

	return oauthClient(accessToken)
}

func oauthClient(accessToken string) *github.Client {
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: accessToken},
	}
	return github.NewClient(t.Client())
}

func getCredentials() (string, string, string) {
	scan := func() string {
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		return strings.TrimSpace(input)
	}

	fmt.Print("Username: ")
	login := scan()
	fmt.Print("Password: ")
	password := strings.TrimSpace(fmt.Sprintf("%s", gopass.GetPasswd()))
	fmt.Print("Two Factor Auth: ")
	tfaToken := scan()

	return login, password, tfaToken
}

// 既に appName という名前のトークンがあるかどうか探す
func findAccessToken(s *napping.Session, appName string) (string, error) {
	type authorization struct {
		ID        int
		URL       string
		Scopes    []string
		Token     string
		App       map[string]string
		Note      string `json:"note"`
		NoteURL   string `json:"note_url"`
		UpdatedAt string `json:"updated_at"`
		CreatedAt string `json:"created_at"`
	}
	res := []authorization{}

	e := struct {
		Message string
		Errors  []struct {
			Resource string
			Field    string
			Code     string
		}
	}{}

	fetchAuthorizations := func(url string) ([]authorization, string, error) {
		resp, err := s.Get(url, nil, &res, &e)
		if err != nil {
			return nil, "", err
		}
		if resp.Status() == 200 {
			linkStr := resp.HttpResponse().Header["Link"][0]
			reg, _ := regexp.Compile("\\<(.*)?\\>; rel=\"next\"")
			subMatch := reg.FindStringSubmatch(linkStr)
			var nextURL string
			if len(subMatch) > 0 {
				nextURL = subMatch[1]
			} else {
				nextURL = ""
			}
			return res, nextURL, nil
		}
		return nil, "", errors.New("Failed to fetch Authentications.")
	}

	authorizations := []authorization{}
	url := "https://api.github.com/authorizations"
	for {
		res, nextURL, err := fetchAuthorizations(url)
		if err != nil {
			return "", nil
		}
		if nextURL == "" {
			break
		}
		url = nextURL
		authorizations = append(authorizations, res...)
	}

	for i := 0; i < len(authorizations); i++ {
		// "hoge (API)" という名前が付けられるようなのでそれで探す
		// FIXME: note 属性で探したかったが、レスポンスについてこなかった
		if authorizations[i].App["name"] == appName+" (API)" {
			return authorizations[i].Token, nil
		}
	}

	return "", nil
}

// 新しくトークンを作る
func createAccessToken(s *napping.Session, appName string, scopes []string) (string, error) {
	res := struct {
		ID        int
		URL       string
		Scopes    []string
		Token     string
		App       map[string]string
		Note      string `json:"note"`
		NoteURL   string `json:"note_url"`
		UpdatedAt string `json:"updated_at"`
		CreatedAt string `json:"created_at"`
	}{}

	e := struct {
		Message string
		Errors  []struct {
			Resource string
			Field    string
			Code     string
		}
	}{}

	payload := struct {
		Scopes []string `json:"scopes"`
		Note   string   `json:"note"`
	}{
		Scopes: scopes,
		Note:   appName,
	}

	url := "https://api.github.com/authorizations"
	resp, err := s.Post(url, &payload, &res, &e)
	if err != nil {
		return "", err
	}
	if resp.Status() == 201 {
		return res.Token, nil
	}

	fmt.Println("Bad response status from Github server")
	fmt.Printf("\t Status:  %v\n", resp.Status())
	fmt.Printf("\t Message: %v\n", e.Message)
	fmt.Printf("\t Errors: %v\n", e.Message)
	return "", errors.New("Failed to create Access Token.")
}

func fetchAccessToken(appName string, scopes []string) (string, error) {
	login, password, tfaToken := getCredentials()

	// Two Factor Auth の認証をするため一度ダミーでリクエストする
	// https://github.com/github/hub/blob/master/lib/hub/github_api.rb#L338
	if tfaToken != "" {
		dummySession := napping.Session{
			Userinfo: url.UserPassword(login, password),
		}
		url := "https://api.github.com/authorizations"
		dummySession.Post(url, nil, nil, nil)
	}

	header := &http.Header{}
	header.Add("X-GitHub-OTP", tfaToken)

	s := napping.Session{
		Userinfo: url.UserPassword(login, password),
		Header:   header,
	}

	foundToken, err := findAccessToken(&s, appName)
	if err != nil {
		return "", err
	}
	if foundToken != "" {
		return foundToken, nil
	}

	createdToken, err := createAccessToken(&s, appName, scopes)
	if err != nil {
		return "", err
	}
	return createdToken, nil
}

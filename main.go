package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {

	url := "https://simple-proxy.xartpvt.workers.dev?destination=https://www.showbox.media/index/share_link?id=36425&type=1"
	method := "GET"

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0")
	req.Header.Add("Cookie", "ci=16764542417a84; sl-session=jxOEJLy+Z2fCZx6oW060FQ==")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}

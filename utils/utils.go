package utils

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	URL "net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/andlabs/ui"
	_ "github.com/andlabs/ui/winmanifest"
	twitterscraper "github.com/n0madic/twitter-scraper"
)

type Opts struct {
	Username     string
	Tweet_id     string
	Batch        string
	Output       string
	Media        string
	Nbr          int
	Dtype        string
	Size         int
	Retweet      bool
	Retweet_only bool
	Proxy        string
	Metadata     bool
}

var (
	Log       *ui.MultilineEntry
	LogSingle *ui.MultilineEntry
	mu        = &sync.Mutex{}
	LogUser   *ui.MultilineEntry
	rt        bool
	GUI       bool
	Stop      = make(chan bool)
	stopOnce  sync.Once
	quality   = map[int]string{
		0: "orig",
		1: "normal",
		2: "small",
	}
	client = &http.Client{
		Timeout: time.Second * 20,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Duration(5) * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: time.Duration(5) * time.Second,
		},
	}
)

var download_id = make(chan string)

// TriggerStop closes the Stop channel (idempotent — safe to call multiple times).
func TriggerStop() {
	stopOnce.Do(func() {
		close(Stop)
	})
}

// ResetStop creates a fresh Stop channel for a new download session.
func ResetStop() {
	Stop = make(chan bool)
	stopOnce = sync.Once{}
}


func LogErr(err string) {
	mu.Lock()
	ui.QueueMain(func() {
		Log.Append(err + "\n")
	})
	mu.Unlock()
}

func LogUserMsg(msg string) {
	if LogUser == nil {
		return
	}
	mu.Lock()
	ui.QueueMain(func() {
		LogUser.Append(msg + "\n")
	})
	mu.Unlock()
}

// logTweetInfo displays tweet metadata (likes, retweets, text, etc.) in the GUI.
func logTweetInfo(tweet *twitterscraper.Tweet, logEntry *ui.MultilineEntry) {
	if logEntry == nil {
		return
	}
	ts := time.Unix(tweet.Timestamp, 0).Format("2006-01-02 15:04:05")
	photoCount := len(tweet.Photos)
	videoCount := len(tweet.Videos)

	msg := fmt.Sprintf(
		"--- @%s (%s) | %s ---\n"+
			"%s\n"+
			"Likes: %d  Retweets: %d  Replies: %d  Views: %d\n"+
			"Media: %d photo(s), %d video(s)\n"+
			"URL: %s\n",
		tweet.Username, tweet.Name, ts,
		tweet.Text,
		tweet.Likes, tweet.Retweets, tweet.Replies, tweet.Views,
		photoCount, videoCount,
		tweet.PermanentURL,
	)

	mu.Lock()
	ui.QueueMain(func() {
		logEntry.Append(msg)
	})
	mu.Unlock()
}

func Name(s string) string {
	segments := strings.Split(s, "/")
	name := segments[len(segments)-1]

	re := regexp.MustCompile(`name=`)
	if re.MatchString(name) {
		segments := strings.Split(name, "?")
		name = segments[len(segments)-2]
	}
	return name
}

func UserTDownload(opt Opts) {
	var wg sync.WaitGroup
	if opt.Proxy != "" {
		proxyURL, _ := URL.Parse(opt.Proxy)
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
	}
	os.MkdirAll(opt.Output+"/"+opt.Username, os.ModePerm)
	scraper := twitterscraper.New()
	scraper.WithReplies(true)
	// do nothing if proxy = ""
	scraper.SetProxy(opt.Proxy)
	for tweet := range scraper.GetTweets(context.Background(), opt.Username, opt.Nbr) {
		select {
		case <-Stop:
			wg.Wait()
			return
		default:
		}

		if tweet.Error != nil {
			LogErr(tweet.Error.Error())
			continue
		}

		if opt.Metadata {
			logTweetInfo(&tweet.Tweet, LogUser)
		}

		// Run synchronously to avoid WaitGroup race (wg.Add must happen before wg.Wait)
		if opt.Media == "videos" || opt.Media == "all" {
			videoUser(&wg, tweet, opt)
		}
		if opt.Media == "pictures" || opt.Media == "all" {
			photoUser(&wg, tweet, opt)
		}
	}
	wg.Wait()
	if GUI {
		LogUserMsg(fmt.Sprintf("Finished: downloaded from @%s", opt.Username))
	} else {
		fmt.Printf("Download finished: %d tweets from @%s\n", opt.Nbr, opt.Username)
	}
	time.Sleep(1 * time.Second)
}

func videoUser(wg *sync.WaitGroup, tweet *twitterscraper.TweetResult, opt Opts) {
	if tweet.IsRetweet && (opt.Retweet || opt.Retweet_only) {
		opt.Tweet_id = tweet.ID
		rtOpt := opt
		rtOpt.Output = opt.Output + "/" + opt.Username
		SingleTDownload(wg, rtOpt, true, false)
		return
	}
	if opt.Retweet_only {
		return
	}
	for _, i := range tweet.Videos {
		select {
		case <-Stop:
			return
		default:
		}
		// Use URL directly instead of fragile fmt.Sprintf parsing
		v := strings.Split(i.URL, "?")[0]
		wg.Add(1)
		go download(wg, v, opt.Output, opt.Username, GUI)
	}
}

func photoUser(wg *sync.WaitGroup, tweet *twitterscraper.TweetResult, opt Opts) {
	if tweet.IsRetweet && (opt.Retweet || opt.Retweet_only) {
		opt.Tweet_id = tweet.ID
		rtOpt := opt
		rtOpt.Output = opt.Output + "/" + opt.Username
		SingleTDownload(wg, rtOpt, true, false)
		return
	}
	if opt.Retweet_only {
		return
	}
	for _, i := range tweet.Photos {
		select {
		case <-Stop:
			return
		default:
		}
		if strings.Contains(i.URL, "video_thumb/") {
			continue
		}
		var url string
		if quality[opt.Size] == "orig" || quality[opt.Size] == "small" {
			url = i.URL + "?name=" + quality[opt.Size]
		} else {
			url = i.URL
		}
		wg.Add(1)
		go download(wg, url, opt.Output, opt.Username, GUI)
	}
}

func BatchTDownload(opt Opts) {
	var wg sync.WaitGroup
	scanner := bufio.NewScanner(strings.NewReader(opt.Batch))
	if opt.Proxy != "" {
		proxyURL, _ := URL.Parse(opt.Proxy)
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		opt.Tweet_id = line
		wg.Add(1)
		go func(o Opts) {
			defer wg.Done()
			SingleTDownload(&wg, o, false, true)
		}(opt)
	}
	wg.Wait()
}

func SingleTDownload(wg *sync.WaitGroup, opt Opts, rt bool, batch bool) {
	if opt.Proxy != "" && (!rt && !batch) {
		proxyURL, _ := URL.Parse(opt.Proxy)
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
	}
	scraper := twitterscraper.New()
	scraper.SetProxy(opt.Proxy)
	tweet, err := scraper.GetTweet(opt.Tweet_id)
	if err != nil {
		LogErr("Scrap: " + err.Error())
		return
	}

	// Show metadata in SingleTweet log (not for retweet processing or batch)
	if GUI && !rt && !batch && opt.Metadata {
		logTweetInfo(tweet, LogSingle)
	}

	// Process videos
	for _, i := range tweet.Videos {
		select {
		case <-Stop:
			return
		default:
		}
		v := strings.Split(i.URL, "?")[0]
		wg.Add(1)
		if GUI {
			go func(url string) {
				download(wg, url, opt.Output, "", true)
				if !rt && !batch {
					n := Name(url)
					mu.Lock()
					ui.QueueMain(func() {
						if LogSingle != nil {
							LogSingle.Append("Downloaded vid: " + n + "\n")
						}
					})
					mu.Unlock()
				}
			}(v)
		} else {
			go download(wg, v, opt.Output, "", false)
		}
	}

	// Process photos
	for _, i := range tweet.Photos {
		select {
		case <-Stop:
			return
		default:
		}
		if strings.Contains(i.URL, "video_thumb/") {
			continue
		}
		var url string
		if quality[opt.Size] == "orig" || quality[opt.Size] == "small" {
			url = i.URL + "?name=" + quality[opt.Size]
		} else {
			url = i.URL
		}
		wg.Add(1)
		if GUI {
			go func(u string) {
				download(wg, u, opt.Output, "", true)
				if !rt && !batch {
					n := Name(u)
					mu.Lock()
					ui.QueueMain(func() {
						if LogSingle != nil {
							LogSingle.Append("Downloaded img: " + n + "\n")
						}
					})
					mu.Unlock()
				}
			}(url)
		} else {
			go download(wg, url, opt.Output, "", false)
		}
	}

	if GUI && !rt && !batch {
		wg.Wait()
		mu.Lock()
		ui.QueueMain(func() {
			if LogSingle != nil {
				LogSingle.Append("--------------------------\n")
			}
		})
		mu.Unlock()
	}
}

// download fetches a media URL and saves it to disk.
// Context cancellation is used to support the Stop channel,
// so there is no deadlock from the old finish-channel approach.
func download(wg *sync.WaitGroup, url string, output string, user string, gui bool) {
	defer wg.Done()

	// Bail out immediately if already stopped
	select {
	case <-Stop:
		return
	default:
	}

	name := Name(url)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel the HTTP request when Stop fires
	go func() {
		select {
		case <-Stop:
			cancel()
		case <-ctx.Done():
		}
	}()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		if gui {
			LogErr("http: " + err.Error())
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64)")

	resp, err := client.Do(req)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			if gui {
				LogErr("http: " + err.Error())
			} else {
				fmt.Println(err.Error())
			}
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}

	var f *os.File
	var ferr error
	if user != "" {
		f, ferr = os.Create(output + "/" + user + "/" + name)
	} else {
		f, ferr = os.Create(output + "/" + name)
	}
	if ferr != nil {
		if gui {
			LogErr("file: " + ferr.Error())
		} else {
			fmt.Println(ferr.Error())
		}
		return
	}
	defer f.Close()

	_, cerr := io.Copy(f, resp.Body)
	if cerr != nil {
		if !errors.Is(cerr, context.Canceled) {
			if gui {
				LogErr("Copy: " + cerr.Error())
			} else {
				fmt.Println("Copy: " + cerr.Error())
			}
		}
		return
	}
	fmt.Println("Download: ", name)
}


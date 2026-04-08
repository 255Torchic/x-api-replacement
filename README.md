# twmd: CLI twitter media downloader (without api key)

This twitter downloader doesn't require Credentials or an api key. It's based on [twitter-scrapper](https://github.com/imperatrona/twitter-scraper).

### Note
For NSFW or private accounts, you will need to logged in (-L). Username and password login isn't supported anymore. You'll have to logged in in a browser and copy auth_token and ct0 cookies (right click => inspect => storage => cookies).
It will create a twmd_cookies.json so you will not have to enter these cookies everytime.


![gui](.github/screenshots/gui.png)

## usage: 

```
Usage:
-h, --help                   Show this help
-u, --user=USERNAME          User you want to download
-t, --tweet=TWEET_ID         Single tweet to download
-n, --nbr=NBR                Number of tweets to download
-i, --img                    Download images only
-v, --video                  Download videos only
-a, --all                    Download images and videos
-r, --retweet                Download retweet too
-z, --url                    Print media url without download it
-R, --retweet-only           Download only retweet
-M, --mediatweet-only        Download only media tweet
-m, --metadata               Save tweet metadata as JSON (likes, retweets, text, media URLs)
-s, --size=SIZE              Choose size between small|normal|large (default
                             large)
-U, --update                 Download missing tweet only
-o, --output=DIR             Output directory
-f, --file-format=FORMAT     Formatted name for the downloaded file, {DATE}
                             {USERNAME} {NAME} {TITLE} {ID}
-d, --date-format=FORMAT     Apply custom date format.
                             (https://go.dev/src/time/format.go)
-L, --login                  Login (needed for NSFW tweets)
-C, --cookies                Use cookies for authentication
-p, --proxy=PROXY            Use proxy (proto://ip:port)
-V, --version                Print version and exit
-B, --no-banner              Don't print banner
```

### Examples:

#### Download 300 tweets from @Spraytrains.

If the tweet doesn't contain a photo or video nothing will be downloaded but it will count towards the 300.

```sh
twmd -u Spraytrains -o ~/Downloads -a -n 300
```

Due to rate limits of twitter, it is possible to fetch at most 500–600 tweets.
To fetch as more tweets as possible, change the argument of `-n` to a bigger number, like 3000.

You can use `-r|--retweet` to download retweets as well, or `-R|--retweet-only` to download retweet only.

`-U|--update` will only download missing media.

#### Download a single tweet with metadata:

```sh
twmd -t 156170319961391104
```

Use `-m|--metadata` to additionally save tweet metadata (likes, retweet count, text, media URLs, etc.) as a JSON file alongside the downloaded media:

```sh
twmd -t 156170319961391104 -m
```

Output: `156170319961391104_metadata.json`

```json
{
  "id": "156170319961391104",
  "username": "example",
  "name": "Example User",
  "text": "Tweet text here",
  "timestamp": "2024-01-01T12:00:00Z",
  "likes": 123,
  "retweets": 45,
  "replies": 6,
  "views": 7890,
  "permanent_url": "https://x.com/example/status/156170319961391104",
  "is_retweet": false,
  "media": [
    { "type": "photo", "url": "https://pbs.twimg.com/media/..." },
    { "type": "video", "url": "https://video.twimg.com/...", "preview": "https://pbs.twimg.com/..." }
  ]
}
```

The `-m` flag also works with user downloads (`-u`) and saves one JSON file per tweet.

#### NSFW tweets

You'll need to login `-L|--login` for downloading nsfw tweets. Or you can provide cookies `-C|--cookies` to complete the login.


#### Using proxy

Both http and socks4/5 can be used:

```sh
twmd  --proxy socks5://127.0.0.1:9050 -t 156170319961391104
```

### GUI

A graphical interface (`twmd-GUI.exe` on Windows, `twmd-GUI` on Linux) is available.  
Each download tab provides:

- **Single Tweet** — download media from one tweet by ID. Check *Show metadata* to display likes, retweet count, replies, views, and media URLs in the log pane.
- **User Download** — download all media from a user's timeline. Supports retweet filtering, picture size selection, and a live log with optional metadata display.
- **Batch Tweet** — paste multiple tweet IDs (one per line) to download them all at once.
- **Errors Log** — centralised error output.

The Stop button in every tab cancels the current download cleanly.

### Installation:

**Note:** If you don't want to build it you can download prebuilt binaries [here](https://github.com/mmpx12/twitter-media-downloader/releases/latest).


#### CLI:

```sh
git clone https://github.com/mmpx12/twitter-media-downloader.git
cd twitter-media-downloader
make
sudo make install
# OR
sudo make all
# Clean
sudo make clean
```

#### GUI:

Requires CGO and GCC (MinGW-w64 on Windows, system GCC on Linux).

```sh
git clone https://github.com/mmpx12/twitter-media-downloader.git
cd twitter-media-downloader
# Linux
make linux-gui
# Windows (requires MinGW-w64 in PATH or set CC= accordingly)
make windows-gui
```

On Windows with MSYS2:
```powershell
$env:CGO_ENABLED="1"
$env:CC="C:\msys64\mingw64\bin\gcc.exe"
$env:PATH="C:\msys64\mingw64\bin;$env:PATH"
go build -o twmd-GUI.exe gui.go
```


#### Termux (no root):

installation: 

```sh
git clone https://github.com/mmpx12/twitter-media-downloader.git
cd twitter-media-downloader
make
make termux-install
# OR
make termux-all
# Clean
make termux-clean
```

You may also want to add stuff in ~/bin/termux-url-opener to automatically download profile or post when share with termux.

```sh
cd ~/storage/downlaods
if grep twitter <<< "$1" >/dev/null; then
  if [[ $(tr -cd '/' <<< "$1" | wc -c) -eq 3 ]]; then
    userid=$(cut -d '/' -f 4 <<< "$1" |  cut -d '?' -f 1)
    echo "$userid"
    twmd -B -u "$userid" -o twitter -i -v -n 3000
  else 
    postid=$(cut -d '/' -f 6 <<< "$1" |  cut -d '?' -f 1)
    twmd -B -t "$postid" -o twitter
  fi
fi
```


Check [here](https://gist.github.com/mmpx12/f0741d40909ed3f182fd6f9b33b580d7) for a full termux-url-opener example.


#### Gifs are not supported at the moment.


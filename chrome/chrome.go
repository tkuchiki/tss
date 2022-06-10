package chrome

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type Chrome struct {
	debug bool
}

type ScreenshotArgs struct {
	URL          string
	Cookie       string
	WaitSelector string
	Filename     string
	Width        int64
	Height       int64
}

func New(debug bool) *Chrome {
	return &Chrome{
		debug: debug,
	}
}

func cookiesToMap(cookies string) map[string]string {
	splitCookies := strings.Split(cookies, ";")
	cookieMap := make(map[string]string, 0)
	for _, cookie := range splitCookies {
		c := strings.Split(cookie, "=")
		cookieMap[strings.TrimSpace(c[0])] = strings.TrimSpace(c[1])
	}

	return cookieMap
}

func (c *Chrome) TakeScreenshot(ctx context.Context, args *ScreenshotArgs) error {
	var buf []byte

	var cookies map[string]string
	if args.Cookie != "" {
		cookies = cookiesToMap(args.Cookie)
	}

	if c.debug {
		log.Println(args.URL)
	}

	if args.Width > 0 || args.Height > 0 {
		if err := chromedp.Run(ctx, takeScreenshot(args, cookies, &buf, c.debug)); err != nil {
			return err
		}
	} else {
		if err := chromedp.Run(ctx, fullScreenshot(args, cookies, &buf, c.debug)); err != nil {
			return err
		}
	}
	if err := ioutil.WriteFile(args.Filename, buf, 0o644); err != nil {
		return err
	}

	return nil
}

func setCookieAction(_url string, cookies map[string]string, debug bool) chromedp.ActionFunc {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		if len(cookies) > 0 {
			u, err := url.Parse(_url)
			if err != nil {
				return err
			}

			// create cookie expiration
			expr := cdp.TimeSinceEpoch(time.Now().Add(1 * time.Hour))
			// add cookies to chrome
			for key, val := range cookies {
				err := network.SetCookie(key, val).
					WithExpires(&expr).
					WithDomain(u.Hostname()).
					WithHTTPOnly(false).
					WithSecure(false).
					Do(ctx)
				if err != nil {
					if debug {
						log.Println(fmt.Errorf("could not set cookie %q to %q", val, key))
					}
				}
			}
		}
		return nil
	})
}

func takeScreenshot(args *ScreenshotArgs, cookies map[string]string, res *[]byte, debug bool) chromedp.Tasks {
	return chromedp.Tasks{
		setCookieAction(args.URL, cookies, debug),
		// Open URL
		chromedp.Navigate(args.URL),
		chromedp.WaitReady(args.WaitSelector),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, _, contentSize, _, _, cssContentSize, err := page.GetLayoutMetrics().Do(ctx)
			if err != nil {
				return err
			}
			if cssContentSize != nil {
				contentSize = cssContentSize
			}

			var widthi int64
			var heighti int64
			var widthf float64
			var heightf float64

			if args.Width != 0 {
				widthi = args.Width
				widthf = float64(widthi)
			} else {
				widthi = args.Width
				widthf = contentSize.Width
			}

			if args.Height != 0 {
				heighti = args.Height
				heightf = float64(heighti)
			} else {
				heighti = args.Height
				heightf = contentSize.Height
			}

			// force viewport emulation
			err = emulation.SetDeviceMetricsOverride(widthi, heighti, 1, false).
				WithScreenOrientation(&emulation.ScreenOrientation{
					Type:  emulation.OrientationTypePortraitPrimary,
					Angle: 0,
				}).
				Do(ctx)
			if err != nil {
				return err
			}

			// capture screenshot
			*res, err = page.CaptureScreenshot().
				WithQuality(100).
				WithClip(&page.Viewport{
					X:      contentSize.X,
					Y:      contentSize.Y,
					Width:  widthf,
					Height: heightf,
					Scale:  1,
				}).Do(ctx)
			if err != nil {
				return err
			}

			return nil
		}),
	}
}

func fullScreenshot(args *ScreenshotArgs, cookies map[string]string, res *[]byte, debug bool) chromedp.Tasks {
	return chromedp.Tasks{
		setCookieAction(args.URL, cookies, debug),
		// Open URL
		chromedp.Navigate(args.URL),
		chromedp.WaitReady(args.WaitSelector),
		chromedp.FullScreenshot(res, 100),
	}
}

// TODO: Fix to can change flags
func NewExecAllocatorOptions(headless, hideScrollbars bool) []chromedp.ExecAllocatorOption {
	// https://peter.sh/experiments/chromium-command-line-switches/
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-client-side-phishing-detection", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-features", "site-per-process,TranslateUI,BlinkGenPropertyTrees"),
		chromedp.Flag("disable-hang-monitor", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("disable-prompt-on-repost", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("hide-scrollbars", hideScrollbars),
		chromedp.Flag("mute-audio", true),
	)

	return opts
}

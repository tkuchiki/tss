package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/chromedp/chromedp"
	"github.com/tkuchiki/tss/chrome"
)

var version string

func readCookie(file string) (string, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

	return string(b), err
}

func run() error {
	var cookieFile string
	var screenshotFile string
	var noHeadless bool
	var hideScrollbars bool
	var url string
	var waitSelector string
	var width int64
	var height int64
	var debug bool
	var printVersion bool

	flag.StringVar(&url, "url", "", "URL")
	flag.StringVar(&cookieFile, "cookie", "", "Cookie file")
	flag.StringVar(&screenshotFile, "screenshot", "screenshot.png", "Screenshot file")
	flag.BoolVar(&noHeadless, "no-headless", false, "Disable headless mode")
	flag.BoolVar(&hideScrollbars, "hide-scrollbars", false, "Hide scrollbar")
	flag.StringVar(&waitSelector, "wait-selector", "body", "CSS selector for chromedp.WaitReady")
	flag.Int64Var(&width, "width", 0, "Width")
	flag.Int64Var(&height, "height", 0, "Height")
	flag.BoolVar(&debug, "debug", false, "Debug mode")
	flag.BoolVar(&printVersion, "version", false, "Print version")

	flag.Parse()

	if printVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	var cookie string
	var err error
	if cookieFile != "" {
		cookie, err = readCookie(cookieFile)
		if err != nil {
			return err
		}
	}

	opts := chrome.NewExecAllocatorOptions(!noHeadless, hideScrollbars)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func(cancel, allocCancel context.CancelFunc) {
		for {
			s := <-sig
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				cancel()
				allocCancel()
				log.Println("stopped chrome")
				os.Exit(0)
			}
		}
	}(cancel, allocCancel)

	browser := chrome.New(debug)

	screenshotArgs := &chrome.ScreenshotArgs{
		URL:          url,
		Filename:     screenshotFile,
		Cookie:       cookie,
		WaitSelector: waitSelector,
		Width:        width,
		Height:       height,
	}

	err = browser.TakeScreenshot(ctx, screenshotArgs)

	return err
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
